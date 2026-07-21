package agentConf

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	yaml "gopkg.in/yaml.v3"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/app/opamp"
	filterprocessor "github.com/SigNoz/signoz/pkg/query-service/app/opamp/otelconfig/filterprocessor"
	tsp "github.com/SigNoz/signoz/pkg/query-service/app/opamp/otelconfig/tailsampler"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/opamptypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

var m *Manager

var (
	CodeConfigVersionNoConfig = errors.MustNewCode("config_version_no_config")
)

func init() {
	m = &Manager{}
}

type Manager struct {
	Repo
	// lock to make sure only one update is sent to remote agents at a time
	lock   uint32
	logger *slog.Logger

	configSubscribers     map[string]func()
	configSubscribersLock sync.Mutex
}

type ManagerOptions struct {
	Store sqlstore.SQLStore
}

func Initiate(options *ManagerOptions) (*Manager, error) {
	m = &Manager{
		Repo:              Repo{options.Store},
		logger:            slog.Default(),
		configSubscribers: map[string]func(){},
	}

	return m, nil
}

// Implements opamp.AgentConfigProvider
func (m *Manager) SubscribeToConfigUpdates(callback func()) (unsubscribe func()) {
	m.configSubscribersLock.Lock()
	defer m.configSubscribersLock.Unlock()

	subscriberId := uuid.NewString()
	m.configSubscribers[subscriberId] = callback

	return func() {
		delete(m.configSubscribers, subscriberId)
	}
}

func (m *Manager) notifyConfigUpdateSubscribers() {
	m.configSubscribersLock.Lock()
	defer m.configSubscribersLock.Unlock()
	for _, handler := range m.configSubscribers {
		handler()
	}
}

// Implements opamp.AgentConfigProvider
func (m *Manager) RecommendAgentConfig(_ valuer.UUID, currentConfYaml []byte) (
	recommendedConfYaml []byte,
	// Opaque id of the recommended config, used for reporting deployment status updates
	configId string,
	err error,
) {
	hash := sha256.New()
	hash.Write(currentConfYaml)

	return currentConfYaml, string(hash.Sum(nil)), nil
}

// Implements opamp.AgentConfigProvider
func (m *Manager) ReportConfigDeploymentStatus(
	orgId valuer.UUID,
	agentId string,
	configId string,
	err error,
) {
	featureConfigIds := strings.Split(configId, ",")
	for _, featureConfId := range featureConfigIds {
		newStatus := opamptypes.Deployed.StringValue()
		message := "Deployment was successful"
		if err != nil {
			newStatus = opamptypes.DeployFailed.StringValue()
			message = fmt.Sprintf("%s: %s", agentId, err.Error())
		}
		_ = m.updateDeployStatusByHash(
			context.Background(), orgId, featureConfId, newStatus, message,
		)
	}
}

// Implements model.AgentConfigProvider
func (m *Manager) GetDeployStatusByHash(ctx context.Context, orgId valuer.UUID, configHash string) (opamptypes.DeployStatus, error) {
	return m.Repo.GetDeployStatusByHash(ctx, orgId, configHash)
}

func GetLatestVersion(
	ctx context.Context, orgId valuer.UUID, elementType opamptypes.ElementType,
) (*opamptypes.AgentConfigVersion, error) {
	return m.GetLatestVersion(ctx, orgId, elementType)
}

func GetConfigVersion(
	ctx context.Context, orgId valuer.UUID, elementType opamptypes.ElementType, version int,
) (*opamptypes.AgentConfigVersion, error) {
	return m.GetConfigVersion(ctx, orgId, elementType, version)
}

func GetConfigHistory(
	ctx context.Context, orgId valuer.UUID, typ opamptypes.ElementType, limit int,
) ([]opamptypes.AgentConfigVersion, error) {
	return m.GetConfigHistory(ctx, orgId, typ, limit)
}

// StartNewVersion launches a new config version for given set of elements
func StartNewVersion(
	ctx context.Context, orgId valuer.UUID, userId valuer.UUID, eleType opamptypes.ElementType, elementIds []string,
) (*opamptypes.AgentConfigVersion, error) {

	// create a new version
	cfg := opamptypes.NewAgentConfigVersion(orgId, userId, eleType)

	// insert new config and elements into database
	err := m.insertConfig(ctx, orgId, userId, cfg, elementIds)
	if err != nil {
		return nil, err
	}

	m.notifyConfigUpdateSubscribers()

	return cfg, nil
}

func NotifyConfigUpdate(ctx context.Context) {
	m.notifyConfigUpdateSubscribers()
}

func Redeploy(ctx context.Context, orgId valuer.UUID, typ opamptypes.ElementType, version int) error {
	configVersion, err := GetConfigVersion(ctx, orgId, typ, version)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch config version during redeploy", errors.Attr(err))
		return err
	}

	if configVersion == nil || (configVersion != nil && configVersion.Config == "") {
		slog.DebugContext(ctx, "config version has no conf yaml", "config_version", configVersion)
		return errors.NewInvalidInputf(CodeConfigVersionNoConfig, "the config version can not be redeployed")
	}
	switch typ {
	case opamptypes.ElementTypeSamplingRules:
		var config *tsp.Config
		if err := yaml.Unmarshal([]byte(configVersion.Config), &config); err != nil {
			slog.DebugContext(ctx, "failed to read last conf correctly", errors.Attr(err))
			return model.BadRequest(fmt.Errorf("failed to read the stored config correctly"))
		}

		// merge current config with new filter params
		processorConf := map[string]interface{}{
			"signoz_tail_sampling": config,
		}

		opamp.AddToTracePipelineSpec("signoz_tail_sampling")
		configHash, err := opamp.UpsertControlProcessors(ctx, "traces", processorConf, m.OnConfigUpdate)
		if err != nil {
			slog.ErrorContext(ctx, "failed to call agent config update for trace processor", errors.Attr(err))
			return errors.WithAdditionalf(err, "failed to deploy the config")
		}

		m.updateDeployStatus(ctx, orgId, opamptypes.ElementTypeSamplingRules, version, opamptypes.DeployInitiated.StringValue(), "Deployment started", configHash, configVersion.Config)
	case opamptypes.ElementTypeDropRules:
		var filterConfig *filterprocessor.Config
		if err := yaml.Unmarshal([]byte(configVersion.Config), &filterConfig); err != nil {
			slog.ErrorContext(ctx, "failed to read last conf correctly", errors.Attr(err))
			return model.InternalError(fmt.Errorf("failed to read the stored config correctly"))
		}
		processorConf := map[string]interface{}{
			"filter": filterConfig,
		}

		opamp.AddToMetricsPipelineSpec("filter")
		configHash, err := opamp.UpsertControlProcessors(ctx, "metrics", processorConf, m.OnConfigUpdate)
		if err != nil {
			slog.ErrorContext(ctx, "failed to call agent config update for trace processor", errors.Attr(err))
			return err
		}

		m.updateDeployStatus(ctx, orgId, opamptypes.ElementTypeSamplingRules, version, opamptypes.DeployInitiated.StringValue(), "Deployment started", configHash, configVersion.Config)
	}

	return nil
}

// UpsertFilterProcessor updates the agent config with new filter processor params
func UpsertFilterProcessor(ctx context.Context, orgId valuer.UUID, version int, config *filterprocessor.Config) error {
	if !atomic.CompareAndSwapUint32(&m.lock, 0, 1) {
		return fmt.Errorf("agent updater is busy")
	}
	defer atomic.StoreUint32(&m.lock, 0)

	// merge current config with new filter params
	// merge current config with new filter params
	processorConf := map[string]interface{}{
		"filter": config,
	}

	opamp.AddToMetricsPipelineSpec("filter")
	configHash, err := opamp.UpsertControlProcessors(ctx, "metrics", processorConf, m.OnConfigUpdate)
	if err != nil {
		slog.ErrorContext(ctx, "failed to call agent config update for trace processor", errors.Attr(err))
		return err
	}

	processorConfYaml, yamlErr := yaml.Marshal(config)
	if yamlErr != nil {
		slog.WarnContext(ctx, "unexpected error while transforming processor config to yaml", errors.Attr(yamlErr))
	}

	m.updateDeployStatus(ctx, orgId, opamptypes.ElementTypeDropRules, version, opamptypes.DeployInitiated.StringValue(), "Deployment started", configHash, string(processorConfYaml))
	return nil
}

// OnConfigUpdate is a callback function passed to opamp server.
// It receives a config hash with error status.  We assume
// successful deployment if no error is received.
// this method is currently expected to be called only once in the lifecycle
// but can be improved in future to accept continuous request status updates from opamp
func (m *Manager) OnConfigUpdate(orgId valuer.UUID, agentId string, hash string, err error) {

	status := opamptypes.Deployed.StringValue()

	message := "Deployment was successful"

	defer func() {
		m.logger.Info(status, "agent_id", agentId, "agent_response", message)
	}()

	if err != nil {
		status = opamptypes.DeployFailed.StringValue()
		message = fmt.Sprintf("%s: %s", agentId, err.Error())
	}

	_ = m.updateDeployStatusByHash(context.Background(), orgId, hash, status, message)
}

// UpsertSamplingProcessor updates the agent config with new filter processor params
func UpsertSamplingProcessor(ctx context.Context, orgId valuer.UUID, version int, config *tsp.Config) error {
	if !atomic.CompareAndSwapUint32(&m.lock, 0, 1) {
		return fmt.Errorf("agent updater is busy")
	}
	defer atomic.StoreUint32(&m.lock, 0)

	// merge current config with new filter params
	processorConf := map[string]interface{}{
		"signoz_tail_sampling": config,
	}

	opamp.AddToTracePipelineSpec("signoz_tail_sampling")
	configHash, err := opamp.UpsertControlProcessors(ctx, "traces", processorConf, m.OnConfigUpdate)
	if err != nil {
		slog.ErrorContext(ctx, "failed to call agent config update for trace processor", errors.Attr(err))
		return err
	}

	processorConfYaml, yamlErr := yaml.Marshal(config)
	if yamlErr != nil {
		slog.WarnContext(ctx, "unexpected error while transforming processor config to yaml", errors.Attr(yamlErr))
	}

	m.updateDeployStatus(ctx, orgId, opamptypes.ElementTypeSamplingRules, version, opamptypes.DeployInitiated.StringValue(), "Deployment started", configHash, string(processorConfYaml))
	return nil
}
