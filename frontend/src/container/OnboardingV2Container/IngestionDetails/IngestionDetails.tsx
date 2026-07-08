import { useCopyToClipboard } from 'react-use';
import { Button, Skeleton, Tooltip, Typography } from 'antd';
import logEvent from 'api/common/logEvent';
import { DOCS_BASE_URL } from 'constants/app';
import { useGetGlobalConfig } from 'hooks/globalConfig/useGetGlobalConfig';
import { useNotifications } from 'hooks/useNotifications';
import { ArrowUpRight, Copy, Info, TriangleAlert } from 'lucide-react';

import './IngestionDetails.styles.scss';

const ONBOARDING_V3_ANALYTICS_EVENTS_MAP = {
	BASE: 'Onboarding V3',
	INGESTION_URL_COPIED: 'Ingestion URL copied',
};

export default function OnboardingIngestionDetails(): JSX.Element {
	const { notifications } = useNotifications();
	const [, handleCopyToClipboard] = useCopyToClipboard();

	const {
		data: globalConfig,
		isLoading: isLoadingGlobalConfig,
		isError: isErrorGlobalConfig,
		error: globalConfigError,
	} = useGetGlobalConfig();

	const handleCopyKey = (text: string): void => {
		handleCopyToClipboard(text);
		notifications.success({
			message: 'Copied to clipboard',
		});
	};

	return (
		<div className="configure-product-ingestion-section-content">
			<div className="ingestion-key-details-section">
				<div className="ingestion-key-details-section-key">
					{isLoadingGlobalConfig ? (
						<div className="skeleton-container">
							<Skeleton.Input active className="skeleton-input" />
						</div>
					) : (
						<div className="ingestion-key-region-details-section">
							<div className="ingestion-region-container">
								<Typography.Text className="ingestion-region-label">
									Ingestion URL
								</Typography.Text>

								{!isErrorGlobalConfig && (
									<Typography.Text className="ingestion-region-value-copy">
										{globalConfig?.data?.ingestion_url}

										<Copy
											size={14}
											className="copy-btn"
											onClick={(): void => {
												logEvent(
													`${ONBOARDING_V3_ANALYTICS_EVENTS_MAP.BASE}: ${ONBOARDING_V3_ANALYTICS_EVENTS_MAP.INGESTION_URL_COPIED}`,
													{},
												);

												const ingestionURL = globalConfig?.data?.ingestion_url;

												if (ingestionURL) {
													handleCopyKey(ingestionURL);
												}
											}}
										/>
									</Typography.Text>
								)}

								{isErrorGlobalConfig && (
									<Tooltip
										rootClassName="ingestion-url-error-tooltip"
										arrow={false}
										title={
											<div className="ingestion-url-error-content">
												<Typography.Text className="ingestion-url-error-code">
													{globalConfigError?.getErrorCode()}
												</Typography.Text>

												<Typography.Text className="ingestion-url-error-message">
													{globalConfigError?.getErrorMessage()}
												</Typography.Text>
											</div>
										}
										placement="topLeft"
									>
										<Button type="text" icon={<TriangleAlert size={14} />} />
									</Tooltip>
								)}
							</div>
						</div>
					)}
				</div>
			</div>

			<div className="ingestion-setup-details-links">
				<Info size={14} />

				<span>
					Find your ingestion URL and learn more about sending data to SigInsight{' '}
					<a
						href={`${DOCS_BASE_URL}/docs/ingestion/signoz-cloud/overview/`}
						target="_blank"
						className="learn-more"
						rel="noreferrer"
					>
						here <ArrowUpRight size={14} />
					</a>
				</span>
			</div>
		</div>
	);
}
