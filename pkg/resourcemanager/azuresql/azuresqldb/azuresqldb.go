// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package azuresqldb

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2015-05-01-preview/sql"
	azuresqlshared "github.com/Azure/azure-service-operator/pkg/resourcemanager/azuresql/azuresqlshared"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
)

type AzureSqlDbManager struct {
}

func NewAzureSqlDbManager() *AzureSqlDbManager {
	return &AzureSqlDbManager{}
}

// GetServer returns a SQL server
func (_ *AzureSqlDbManager) GetServer(ctx context.Context, resourceGroupName string, serverName string) (result sql.Server, err error) {
	serversClient, err := azuresqlshared.GetGoServersClient()
	if err != nil {
		return sql.Server{}, err
	}

	return serversClient.Get(
		ctx,
		resourceGroupName,
		serverName,
	)
}

// GetDB retrieves a database
func (_ *AzureSqlDbManager) GetDB(ctx context.Context, resourceGroupName string, serverName string, databaseName string) (sql.Database, error) {
	dbClient, err := azuresqlshared.GetGoDbClient()
	if err != nil {
		return sql.Database{}, err
	}

	return dbClient.Get(
		ctx,
		resourceGroupName,
		serverName,
		databaseName,
		"serviceTierAdvisors, transparentDataEncryption",
	)
}

// DeleteDB deletes a DB
func (sdk *AzureSqlDbManager) DeleteDB(ctx context.Context, resourceGroupName string, serverName string, databaseName string) (result autorest.Response, err error) {
	result = autorest.Response{
		Response: &http.Response{
			StatusCode: 200,
		},
	}

	// check to see if the server exists, if it doesn't then short-circuit
	server, err := sdk.GetServer(ctx, resourceGroupName, serverName)
	if err != nil || *server.State != "Ready" {
		return result, nil
	}

	// check to see if the db exists, if it doesn't then short-circuit
	_, err = sdk.GetDB(ctx, resourceGroupName, serverName, databaseName)
	if err != nil {
		return result, nil
	}

	dbClient, err := azuresqlshared.GetGoDbClient()
	if err != nil {
		return result, err
	}

	result, err = dbClient.Delete(
		ctx,
		resourceGroupName,
		serverName,
		databaseName,
	)

	return result, err
}

// CreateOrUpdateDB creates or updates a DB in Azure
func (_ *AzureSqlDbManager) CreateOrUpdateDB(ctx context.Context, resourceGroupName string, location string, serverName string, tags map[string]*string, properties azuresqlshared.SQLDatabaseProperties) (*http.Response, error) {
	dbClient, err := azuresqlshared.GetGoDbClient()
	if err != nil {
		return &http.Response{
			StatusCode: 0,
		}, err
	}

	dbProp := azuresqlshared.SQLDatabasePropertiesToDatabase(properties)

	future, err := dbClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		serverName,
		properties.DatabaseName,
		sql.Database{
			Location:           to.StringPtr(location),
			DatabaseProperties: &dbProp,
			Tags:               tags,
		})

	return future.Response(), err
}

// AddLongTermRetention enables / disables long term retention
func (_ *AzureSqlDbManager) AddLongTermRetention(ctx context.Context, resourceGroupName string, serverName string, databaseName string, enabledDisabled string) (*http.Response, error) {

	var state sql.BackupLongTermRetentionPolicyState
	if strings.EqualFold(enabledDisabled, "enabled") {
		state = sql.Enabled
	} else if strings.EqualFold(enabledDisabled, "disabled") {
		state = sql.Disabled
	} else {
		return &http.Response{
			StatusCode: 0,
		}, fmt.Errorf("State for LongTermRetentionPolicy must be enabled or disabled (or empty)")
	}

	longTermClient, err := azuresqlshared.GetBackupLongTermRetentionPoliciesClient()
	if err != nil {
		return &http.Response{
			StatusCode: 0,
		}, err
	}

	future, err := longTermClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		serverName,
		databaseName,
		sql.BackupLongTermRetentionPolicy{
			BackupLongTermRetentionPolicyProperties: &sql.BackupLongTermRetentionPolicyProperties{
				State: state,
			},
		},
	)

	return future.Response(), err
}
