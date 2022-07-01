package testutils

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/wso2/product-apim-tooling/import-export-cli/integration/apim"
	"github.com/wso2/product-apim-tooling/import-export-cli/integration/base"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
	"os"
	"strings"
	"testing"
)

const (
	PolicyIDKey   = "policyId"
	PolicyNameKey = "policyName"
	PolicyIDReset = "-1"
)

func ValidateThrottlePolicyExportImport(t *testing.T, args *ThrottlePolicyImportExportTestArgs, policyType string) {
	t.Helper()

	// Setup apictl envs
	base.SetupEnv(t, args.SrcAPIM.GetEnvName(), args.SrcAPIM.GetApimURL(), args.SrcAPIM.GetTokenURL())
	base.SetupEnv(t, args.DestAPIM.GetEnvName(), args.DestAPIM.GetApimURL(), args.DestAPIM.GetTokenURL())

	// Export policy from env 1
	base.Login(t, args.SrcAPIM.GetEnvName(), args.CtlUser.Username, args.CtlUser.Password)
	policyName := fmt.Sprintf("%v", args.Policy[PolicyNameKey])

	exportedOutput, _ := exportThrottlePolicy(t, policyName, args.SrcAPIM.GetEnvName())
	args.ImportFilePath = base.GetExportedPathFromOutput(exportedOutput)
	assert.True(t, base.IsFileAvailable(t, args.ImportFilePath))
	args.SrcAPIM.DeleteThrottlePolicy(fmt.Sprintf("%v", args.Policy[PolicyIDKey]), policyType)

	// Import Throttling Policy to env 2
	base.Login(t, args.DestAPIM.GetEnvName(), args.CtlUser.Username, args.CtlUser.Password)
	_, err := importThrottlePolicy(t, args)
	assert.Nil(t, err, "Error while importing the Throttling Policy")
	// Give time for newly imported Throttling Policy to get indexed
	base.WaitForIndexing()

	// Get Throttle Policy from env 2
	//base.Login(t, args.DestAPIM.GetEnvName(), args.Admin.Username, args.Admin.Password)
	importedPolicy, _ := getThrottlingPolicyByName(t, args, policyName, policyType)
	// Validate env 1 and env 2 policy is equal
	ValidatePoliciesEqual(t, args, importedPolicy)
	RemoveExportedThrottlingPolicyFile(t, args.ImportFilePath)
}

func AddNewThrottlePolicy(t *testing.T, client *apim.Client, username, password, policyType string) interface{} {
	client.Login(username, password)
	generatedPolicy := client.GenerateSampleThrottlePolicyData(policyType)
	addedPolicy := client.AddThrottlePolicy(t, generatedPolicy, policyType)
	return addedPolicy
}

func exportThrottlePolicy(t *testing.T, name, env string) (string, error) {
	var output string
	var err error
	output, err = base.Execute(t, "export", "policy", "rate-limiting", "-n", name, "-e", env, "-k", "--verbose")
	return output, err
}

func importThrottlePolicy(t *testing.T, args *ThrottlePolicyImportExportTestArgs) (string, error) {
	output, err := base.Execute(t, "import", "policy", "rate-limiting", "-e", args.DestAPIM.GetEnvName(), "-f", args.ImportFilePath, "-u")
	return output, err
}

func getThrottlingPolicyByName(t *testing.T, args *ThrottlePolicyImportExportTestArgs, throttlePolicyName, policyType string) (map[string]interface{}, error) {
	client := args.DestAPIM
	uuid := client.GetThrottlePolicyID(t, args.Admin.Username, args.Admin.Password, throttlePolicyName, policyType)
	policy := client.GetThrottlePolicy(uuid, policyType)
	client.DeleteThrottlePolicy(uuid, policyType)
	return ThrottlePolicyStructToMap(policy)
}

func ValidatePoliciesEqual(t *testing.T, args *ThrottlePolicyImportExportTestArgs, importedPolicy map[string]interface{}) {
	exportedPolicy := args.Policy
	exportedPolicy[PolicyIDKey] = PolicyIDReset
	importedPolicy[PolicyIDKey] = PolicyIDReset
	assert.Equal(t, exportedPolicy, importedPolicy)
}

func ThrottlePolicyStructToMap(policy interface{}) (map[string]interface{}, error) {
	var throttlePolicy map[string]interface{}
	marshalled, _ := json.Marshal(policy)
	err := json.Unmarshal(marshalled, &throttlePolicy)
	return throttlePolicy, err
}

func RemoveExportedThrottlingPolicyFile(t *testing.T, file string) {
	t.Log("base.RemoveExportedThrottlingPolicyFile() - file path:", file)
	if _, err := os.Stat(file); err == nil {
		err := os.Remove(file)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func ValidateThrottlePoliciesList(t *testing.T, args *ThrottlePolicyImportExportTestArgs) {
	t.Helper()

	// Setup apictl envs
	base.SetupEnv(t, args.SrcAPIM.GetEnvName(), args.SrcAPIM.GetApimURL(), args.SrcAPIM.GetTokenURL())

	// List APIs of env 1
	base.Login(t, args.SrcAPIM.GetEnvName(), args.CtlUser.Username, args.CtlUser.Password)

	base.WaitForIndexing()

	output, _ := listThrottlePolicies(t, args)

	throttlePoliciesList := args.SrcAPIM.GetThrottlePolicies(t, args.CtlUser.Username, args.CtlUser.Password)

	ValidateListThrottlePoliciesEqual(t, output, throttlePoliciesList)
}

func listThrottlePolicies(t *testing.T, args *ThrottlePolicyImportExportTestArgs) (string, error) {
	output, err := base.Execute(t, "get", "policies", "rate-limiting", "-e", args.SrcAPIM.EnvName, "-k", "--verbose")
	return output, err
}

func ValidateListThrottlePoliciesEqual(t *testing.T, throttlePoliciesListOutput string, throttlePoliciesList *utils.PolicyList) {
	unmatchedCount := throttlePoliciesList.Count
	for _, policy := range throttlePoliciesList.List {
		// If the output string contains the same Policy ID, then decrement the count
		assert.Truef(t, strings.Contains(throttlePoliciesListOutput, policy.Uuid), "throttlePoliciesListFromCtl: "+throttlePoliciesListOutput+
			" , does not contain policy.uuid: "+policy.Uuid)
		unmatchedCount--
	}
	// Count == 0 means that all the policies from throttlePoliciesList were in throttlePoliciesListOutput
	assert.Equal(t, 0, unmatchedCount, "Throttle policies lists are not equal")
}
