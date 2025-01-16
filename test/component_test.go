package test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/aws-component-helper"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponent(t *testing.T) {
	t.Parallel()
	// Define the AWS region to use for the tests
	awsRegion := "us-east-2"

	// Initialize the test fixture
	fixture := helper.NewFixture(t, "../", awsRegion, "test/fixtures")

	// Ensure teardown is executed after the test
	defer fixture.TearDown()
	fixture.SetUp(&atmos.Options{})

	// Define the test suite
	fixture.Suite("default", func(t *testing.T, suite *helper.Suite) {
		// t.Parallel()
		suite.AddDependency("vpc", "default-test")

		// Setup phase: Create DNS zones and postgresql for testing
		suite.Setup(t, func(t *testing.T, atm *helper.Atmos) {
			// Deploy the delegated DNS zone
			inputs := map[string]interface{}{
				"zone_config": []map[string]interface{}{
					{
						"subdomain": suite.GetRandomIdentifier(),
						"zone_name": "components.cptest.test-automation.app",
					},
				},
			}
			atm.GetAndDeploy("dns-delegated", "default-test", inputs)
			atm.GetAndDeploy("aurora-postgres", "default-test", map[string]interface{}{
				"cluster_name": suite.GetRandomIdentifier(),
			})
		})

		// Teardown phase: Destroy the DNS zones and postgresql created during setup
		suite.TearDown(t, func(t *testing.T, atm *helper.Atmos) {
			atm.GetAndDestroy("aurora-postgres", "default-test", map[string]interface{}{})
			// Deploy the delegated DNS zone
			inputs := map[string]interface{}{
				"zone_config": []map[string]interface{}{
					{
						"subdomain": suite.GetRandomIdentifier(),
						"zone_name": "components.cptest.test-automation.app",
					},
				},
			}
			atm.GetAndDestroy("dns-delegated", "default-test", inputs)
		})

		// Test phase: Validate the functionality of the component
		suite.Test(t, "database", func(t *testing.T, atm *helper.Atmos) {
			databaseName := strings.ToLower(random.UniqueId())

			inputs := map[string]interface{}{
				"additional_databases": []string{databaseName},
			}

			defer atm.GetAndDestroy("aurora-postgres-resources/database", "default-test", inputs)
			component := atm.GetAndDeploy("aurora-postgres-resources/database", "default-test", inputs)
			assert.NotNil(t, component)

			clusterComponent := helper.NewAtmosComponent("aurora-postgres", "default-test", map[string]interface{}{})
			configMap := map[string]interface{}{}
			atm.OutputStruct(clusterComponent, "config_map", &configMap)

			passwordSSMKey, ok := configMap["password_ssm_key"].(string)
			assert.True(t, ok, "password_ssm_key should be a string")

			adminUsername, ok := configMap["username"].(string)
			assert.True(t, ok, "username should be an string")

			adminUserPassword := aws.GetParameter(t, awsRegion, passwordSSMKey)

			dbUrl, ok := configMap["hostname"].(string)
			assert.True(t, ok, "hostname should be a string")

			dbPort, ok := configMap["port"].(float64)
			assert.True(t, ok, "database_port should be an int")

			schemaExistsInRdsInstance := GetWhetherDatabaseExistsInRdsPostgresInstance(t, dbUrl, int32(dbPort), adminUsername, adminUserPassword, databaseName)
			assert.True(t, schemaExistsInRdsInstance)
		})

		suite.Test(t, "schema", func(t *testing.T, atm *helper.Atmos) {
			schemaName := strings.ToLower(random.UniqueId())
			inputs := map[string]interface{}{
				"additional_schemas": map[string]interface{}{
					schemaName: map[string]interface{}{
						"database": "postgres",
					},
				},
			}
			defer atm.GetAndDestroy("aurora-postgres-resources/schema", "default-test", inputs)
			component := atm.GetAndDeploy("aurora-postgres-resources/schema", "default-test", inputs)
			assert.NotNil(t, component)

			clusterComponent := helper.NewAtmosComponent("aurora-postgres", "default-test", map[string]interface{}{})
			configMap := map[string]interface{}{}
			atm.OutputStruct(clusterComponent, "config_map", &configMap)

			passwordSSMKey, ok := configMap["password_ssm_key"].(string)
			assert.True(t, ok, "password_ssm_key should be a string")

			adminUsername, ok := configMap["username"].(string)
			assert.True(t, ok, "username should be an string")

			adminUserPassword := aws.GetParameter(t, awsRegion, passwordSSMKey)

			dbUrl, ok := configMap["hostname"].(string)
			assert.True(t, ok, "hostname should be a string")

			dbPort, ok := configMap["port"].(float64)
			assert.True(t, ok, "database_port should be an int")

			schemaExistsInRdsInstance := GetWhetherSchemaExistsInRdsPostgresInstance(t, dbUrl, int32(dbPort), adminUsername, adminUserPassword, "postgres", schemaName)
			assert.True(t, schemaExistsInRdsInstance)
		})

		suite.Test(t, "user", func(t *testing.T, atm *helper.Atmos) {
			userName := strings.ToLower(random.UniqueId())
			serviceName := strings.ToLower(random.UniqueId())
			inputs := map[string]interface{}{
				"additional_users": map[string]interface{}{
					serviceName: map[string]interface{}{
						"db_user":     userName,
						"db_password": "",
						"grants": []map[string]interface{}{
							{
								"grant":       []string{"ALL"},
								"db":          "postgres",
								"object_type": "database",
								"schema":      "",
							},
						},
					},
				},
			}
			defer atm.GetAndDestroy("aurora-postgres-resources/user", "default-test", inputs)
			component := atm.GetAndDeploy("aurora-postgres-resources/user", "default-test", inputs)
			assert.NotNil(t, component)

			clusterComponent := helper.NewAtmosComponent("aurora-postgres", "default-test", map[string]interface{}{})
			configMap := map[string]interface{}{}
			atm.OutputStruct(clusterComponent, "config_map", &configMap)

			clusterIdenitfier := atm.Output(clusterComponent, "cluster_identifier")

			passwordSSMKey := fmt.Sprintf("/aurora-postgres/%s/%s/passwords/%s", clusterIdenitfier, serviceName, userName)
			userPassword := aws.GetParameter(t, awsRegion, passwordSSMKey)

			dbUrl, ok := configMap["hostname"].(string)
			assert.True(t, ok, "hostname should be a string")

			dbPort, ok := configMap["port"].(float64)
			assert.True(t, ok, "database_port should be an int")

			schemaExistsInRdsInstance := GetWhetherDatabaseExistsInRdsPostgresInstance(t, dbUrl, int32(dbPort), userName, userPassword, "postgres")
			assert.True(t, schemaExistsInRdsInstance)
		})

		suite.Test(t, "grant", func(t *testing.T, atm *helper.Atmos) {
			t.Skip("Additional grants not working. Read more https://github.com/cloudposse-terraform-components/aws-aurora-postgres-resources/issues/17")
			userName := strings.ToLower(random.UniqueId())
			serviceName := strings.ToLower(random.UniqueId())
			inputs := map[string]interface{}{
				"additional_users": map[string]interface{}{
					serviceName: map[string]interface{}{
						"db_user":     userName,
						"db_password": "",
						"grants":      []map[string]interface{}{},
					},
				},
			}
			defer atm.GetAndDestroy("aurora-postgres-resources/grant", "default-test", inputs)
			component := atm.GetAndDeploy("aurora-postgres-resources/grant", "default-test", inputs)
			assert.NotNil(t, component)

			clusterComponent := helper.NewAtmosComponent("aurora-postgres", "default-test", map[string]interface{}{})
			configMap := map[string]interface{}{}
			atm.OutputStruct(clusterComponent, "config_map", &configMap)

			clusterIdenitfier := atm.Output(clusterComponent, "cluster_identifier")

			passwordSSMKey := fmt.Sprintf("/aurora-postgres/%s/%s/passwords/%s", clusterIdenitfier, serviceName, userName)
			userPassword := aws.GetParameter(t, awsRegion, passwordSSMKey)

			dbUrl, ok := configMap["hostname"].(string)
			assert.True(t, ok, "hostname should be a string")

			dbPort, ok := configMap["port"].(float64)
			assert.True(t, ok, "database_port should be an int")

			component.Vars["additional_grants"] = map[string]interface{}{
				userName: []map[string]interface{}{
					{
						"grant": []string{"ALL"},
						"db":    "postgres",
					},
				},
			}

			atm.Deploy(component)

			grantsExistsInRdsInstance := GetWhetherGrantsExistsInRdsPostgresInstance(t, dbUrl, int32(dbPort), userName, userPassword, "postgres", "public")
			assert.True(t, grantsExistsInRdsInstance)
		})

	})
}

func GetWhetherDatabaseExistsInRdsPostgresInstance(t *testing.T, dbUrl string, dbPort int32, dbUsername string, dbPassword string, databaseName string) bool {
	output, err := GetWhetherDatabaseExistsInRdsPostgresInstanceE(t, dbUrl, dbPort, dbUsername, dbPassword, databaseName)
	require.NoError(t, err)
	return output
}

func GetWhetherDatabaseExistsInRdsPostgresInstanceE(t *testing.T, dbUrl string, dbPort int32, dbUsername string, dbPassword string, databaseName string) (bool, error) {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", dbUrl, dbPort, dbUsername, dbPassword, databaseName)

	db, connErr := sql.Open("pgx", connectionString)
	if connErr != nil {
		return false, connErr
	}
	defer db.Close()
	return true, nil
}

func GetWhetherSchemaExistsInRdsPostgresInstance(t *testing.T, dbUrl string, dbPort int32, dbUsername string, dbPassword string, databaseName string, expectedSchemaName string) bool {
	output, err := GetWhetherSchemaExistsInRdsPostgresInstanceE(t, dbUrl, dbPort, dbUsername, dbPassword, databaseName, expectedSchemaName)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func GetWhetherSchemaExistsInRdsPostgresInstanceE(t *testing.T, dbUrl string, dbPort int32, dbUsername string, dbPassword string, databaseName string, expectedSchemaName string) (bool, error) {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", dbUrl, dbPort, dbUsername, dbPassword, databaseName)

	db, connErr := sql.Open("pgx", connectionString)
	if connErr != nil {
		return false, connErr
	}
	defer db.Close()
	var (
		schemaName string
	)
	sqlStatement := `SELECT "schema_name" FROM "information_schema"."schemata" where schema_name=$1`
	row := db.QueryRow(sqlStatement, expectedSchemaName)
	scanErr := row.Scan(&schemaName)
	if scanErr != nil {
		return false, scanErr
	}
	return true, nil
}

func GetWhetherGrantsExistsInRdsPostgresInstance(t *testing.T, dbUrl string, dbPort int32, dbUsername string, dbPassword string, databaseName string, expectedSchemaName string) bool {
	output, err := GetWhetherGrantsExistsInRdsPostgresInstanceE(t, dbUrl, dbPort, dbUsername, dbPassword, databaseName, expectedSchemaName)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func GetWhetherGrantsExistsInRdsPostgresInstanceE(t *testing.T, dbUrl string, dbPort int32, dbUsername string, dbPassword string, databaseName string, expectedSchemaName string) (bool, error) {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", dbUrl, dbPort, dbUsername, dbPassword, databaseName)

	db, connErr := sql.Open("pgx", connectionString)
	if connErr != nil {
		return false, connErr
	}
	defer db.Close()
	var (
		schemaName string
	)
	sqlStatement := `SELECT grantee AS user, CONCAT(table_schema, '.', table_name) AS table,
			CASE
				WHEN COUNT(privilege_type) = 7 THEN 'ALL'
				ELSE ARRAY_TO_STRING(ARRAY_AGG(privilege_type), ', ')
			END AS grants
		FROM information_schema.role_table_grants
		WHERE grantee = '$1'
		GROUP BY table_name, table_schema, grantee;`
	row := db.QueryRow(sqlStatement, dbUsername)
	scanErr := row.Scan(&schemaName)
	if scanErr != nil {
		return false, scanErr
	}
	return true, nil
}
