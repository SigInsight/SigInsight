package openfgaserver

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/authz"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlstoretest"
)

func TestSharedSQLDatastoreCloseDoesNotCloseBorrowedDatabase(t *testing.T) {
	store := sqlstoretest.New(sqlstore.Config{Provider: "sqlite"}, sqlmock.QueryMatcherEqual)
	datastore, err := NewSQLStore(store, authz.Config{})
	require.NoError(t, err)

	datastore.Close()

	store.Mock().ExpectBegin()
	tx, err := store.SQLDB().BeginTx(context.Background(), nil)
	require.NoError(t, err)
	store.Mock().ExpectRollback()
	require.NoError(t, tx.Rollback())
	require.NoError(t, store.Mock().ExpectationsWereMet())
}
