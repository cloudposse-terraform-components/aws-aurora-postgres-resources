package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	awshelper "github.com/cloudposse/test-helpers/pkg/aws"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
)

type ComponentSuite struct {
	helper.TestSuite
}

func (s *ComponentSuite) TestDatabase() {
	const component = "aurora-postgres-resources/database"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	databaseName := strings.ToLower(random.UniqueId())
	inputs := map[string]interface{}{
		"additional_databases": []string{databaseName},
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	clusterComponent := s.GetAtmosOptions("aurora-postgres", "default-test", nil)
	configMap := map[string]interface{}{}
	atmos.OutputStruct(s.T(), clusterComponent, "config_map", &configMap)

	passwordSSMKey, ok := configMap["password_ssm_key"].(string)
	assert.True(s.T(), ok, "password_ssm_key should be a string")

	adminUsername, ok := configMap["username"].(string)
	assert.True(s.T(), ok, "username should be an string")

	adminUserPassword := aws.GetParameter(s.T(), awsRegion, passwordSSMKey)

	dbUrl, ok := configMap["hostname"].(string)
	assert.True(s.T(), ok, "hostname should be a string")

	dbPort, ok := configMap["port"].(float64)
	assert.True(s.T(), ok, "database_port should be an int")

	schemaExistsInRdsInstance := awshelper.GetWhetherDatabaseExistsInRdsPostgresInstance(s.T(), dbUrl, int32(dbPort), adminUsername, adminUserPassword, databaseName)
	assert.True(s.T(), schemaExistsInRdsInstance)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestSchema() {
	const component = "aurora-postgres-resources/schema"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	schemaName := strings.ToLower(random.UniqueId())
	inputs := map[string]interface{}{
		"additional_schemas": map[string]interface{}{
			schemaName: map[string]interface{}{
				"database": "postgres",
			},
		},
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	clusterComponent := s.GetAtmosOptions("aurora-postgres", "default-test", nil)
	configMap := map[string]interface{}{}
	atmos.OutputStruct(s.T(), clusterComponent, "config_map", &configMap)

	passwordSSMKey, ok := configMap["password_ssm_key"].(string)
	assert.True(s.T(), ok, "password_ssm_key should be a string")

	adminUsername, ok := configMap["username"].(string)
	assert.True(s.T(), ok, "username should be an string")

	adminUserPassword := aws.GetParameter(s.T(), awsRegion, passwordSSMKey)

	dbUrl, ok := configMap["hostname"].(string)
	assert.True(s.T(), ok, "hostname should be a string")

	dbPort, ok := configMap["port"].(float64)
	assert.True(s.T(), ok, "database_port should be an int")

	schemaExistsInRdsInstance := awshelper.GetWhetherSchemaExistsInRdsPostgresInstance(s.T(), dbUrl, int32(dbPort), adminUsername, adminUserPassword, "postgres", schemaName)
	assert.True(s.T(), schemaExistsInRdsInstance)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestUser() {
	const component = "aurora-postgres-resources/user"
	const stack = "default-test"
	const awsRegion = "us-east-2"

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

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	clusterComponent := s.GetAtmosOptions("aurora-postgres", "default-test", nil)
	configMap := map[string]interface{}{}
	atmos.OutputStruct(s.T(), clusterComponent, "config_map", &configMap)

	clusterIdenitfier := atmos.Output(s.T(), clusterComponent, "cluster_identifier")

	passwordSSMKey := fmt.Sprintf("/aurora-postgres/%s/%s/passwords/%s", clusterIdenitfier, serviceName, userName)
	userPassword := aws.GetParameter(s.T(), awsRegion, passwordSSMKey)

	dbUrl, ok := configMap["hostname"].(string)
	assert.True(s.T(), ok, "hostname should be a string")

	dbPort, ok := configMap["port"].(float64)
	assert.True(s.T(), ok, "database_port should be an int")

	schemaExistsInRdsInstance := awshelper.GetWhetherDatabaseExistsInRdsPostgresInstance(s.T(), dbUrl, int32(dbPort), userName, userPassword, "postgres")
	assert.True(s.T(), schemaExistsInRdsInstance)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestGrant() {
	const component = "aurora-postgres-resources/grant"
	const stack = "default-test"
	const awsRegion = "us-east-2"

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

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	clusterComponent := s.GetAtmosOptions("aurora-postgres", "default-test", nil)
	configMap := map[string]interface{}{}
	atmos.OutputStruct(s.T(), clusterComponent, "config_map", &configMap)

	clusterIdenitfier := atmos.Output(s.T(), clusterComponent, "cluster_identifier")

	passwordSSMKey := fmt.Sprintf("/aurora-postgres/%s/%s/passwords/%s", clusterIdenitfier, serviceName, userName)
	userPassword := aws.GetParameter(s.T(), awsRegion, passwordSSMKey)

	dbUrl, ok := configMap["hostname"].(string)
	assert.True(s.T(), ok, "hostname should be a string")

	dbPort, ok := configMap["port"].(float64)
	assert.True(s.T(), ok, "database_port should be an int")

	inputs["additional_grants"] = map[string]interface{}{
		userName: []map[string]interface{}{
			{
				"grant": []string{"ALL"},
				"db":    "postgres",
			},
		},
	}

	options, _ = s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	grantsExistsInRdsInstance := awshelper.GetWhetherGrantsExistsInRdsPostgresInstance(s.T(), dbUrl, int32(dbPort), userName, userPassword, "postgres", "public")
	assert.True(s.T(), grantsExistsInRdsInstance)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestDisabled() {
	const component = "aurora-postgres-resources/disabled"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	s.VerifyEnabledFlag(component, stack, nil)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)

	suite.AddDependency(t, "vpc", "default-test", nil)

	subdomain := strings.ToLower(random.UniqueId())
	dnsDelegatedInputs := map[string]interface{}{
		"zone_config": []map[string]interface{}{
			{
				"subdomain": subdomain,
				"zone_name": "components.cptest.test-automation.app",
			},
		},
	}
	suite.AddDependency(t, "dns-delegated", "default-test", &dnsDelegatedInputs)

	clusterName := strings.ToLower(random.UniqueId())
	auroraPostgresInputs := map[string]interface{}{
		"cluster_name": clusterName,
	}
	suite.AddDependency(t, "aurora-postgres", "default-test", &auroraPostgresInputs)
	helper.Run(t, suite)
}

