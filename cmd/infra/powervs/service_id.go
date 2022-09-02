package powervs

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"strings"

	credreqv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	cco "github.com/openshift/cloud-credential-operator/pkg/cmd/provisioning/ibmcloud"
	ccoibmcloud "github.com/openshift/cloud-credential-operator/pkg/ibmcloud"
)

var kubeCloudControllerManagerCR = `
apiVersion: cloudcredential.openshift.io/v1
kind: CredentialsRequest
metadata:
  annotations:
    include.release.openshift.io/self-managed-high-availability: "true"
    include.release.openshift.io/single-node-developer: "true"
  labels:
    controller-tools.k8s.io: "1.0"
  name: openshift-powervs-cloud-controller-manager
  namespace: openshift-cloud-credential-operator
spec:
  providerSpec:
    apiVersion: cloudcredential.openshift.io/v1
    kind: IBMCloudPowerVSProviderSpec
    policies:
    - attributes:
      - name: resourceType
        value: resource-group
      roles:
      - crn:v1:bluemix:public:iam::::role:Viewer
    - attributes:
      - name: serviceName
        value: is
      roles:
      - crn:v1:bluemix:public:iam::::role:Editor
      - crn:v1:bluemix:public:iam::::role:Operator
      - crn:v1:bluemix:public:iam::::role:Viewer
    - attributes:
      - name: serviceName
        value: power-iaas
      roles:
      - crn:v1:bluemix:public:iam::::role:Viewer
      - crn:v1:bluemix:public:iam::::serviceRole:Reader
      - crn:v1:bluemix:public:iam::::serviceRole:Manager
`
var nodePoolManagementCR = `
apiVersion: cloudcredential.openshift.io/v1
kind: CredentialsRequest
metadata:
  name: openshift-capi-powervs
  namespace: openshift-cloud-credential-operator
spec:
  providerSpec:
    apiVersion: cloudcredential.openshift.io/v1
    kind: IBMCloudPowerVSProviderSpec
    policies:
    - attributes:
      - name: serviceName
        value: power-iaas
      roles:
      - crn:v1:bluemix:public:iam::::serviceRole:Manager
      - crn:v1:bluemix:public:iam::::role:Editor
`

var ingressOperatorCR = `
apiVersion: cloudcredential.openshift.io/v1
kind: CredentialsRequest
metadata:
  name: openshift-ingress-powervs
  namespace: openshift-cloud-credential-operator
spec:
  providerSpec:
    apiVersion: cloudcredential.openshift.io/v1
    kind: IBMCloudPowerVSProviderSpec
    policies:
    - attributes:
      - name: serviceName
        value: internet-svcs
      roles:
      - crn:v1:bluemix:public:iam::::serviceRole:Manager
      - crn:v1:bluemix:public:iam::::role:Editor
`

var storageOperatorCR = `
apiVersion: cloudcredential.openshift.io/v1
kind: CredentialsRequest
metadata:
  name: openshift-storage-powervs
  namespace: openshift-cloud-credential-operator
spec:
  providerSpec:
    apiVersion: cloudcredential.openshift.io/v1
    kind: IBMCloudPowerVSProviderSpec
    policies:
    - attributes:
      - name: serviceName
        value: power-iaas
      roles:
      - crn:v1:bluemix:public:iam::::serviceRole:Manager
      - crn:v1:bluemix:public:iam::::role:Editor
    - attributes:
      - name: resourceType
        value: resource-group
      roles:
      - crn:v1:bluemix:public:iam::::role:Viewer
`

// createServiceIDClient creates cloud credential operator's serviceID client
func createServiceIDClient(name, APIKey, accountID, resourceGroupID, crYaml, secretRefName, secretRefNamespace string) (*cco.ServiceID, error) {
	ccoIBMClient, err := ccoibmcloud.NewClient(APIKey, &ccoibmcloud.ClientParams{InfraName: name})
	if err != nil {
		return nil, err
	}

	cr := &credreqv1.CredentialsRequest{}
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(crYaml), 4096)
	if err = decoder.Decode(cr); err != nil {
		return nil, fmt.Errorf("failed to decode to CredentialsRequest %w", err)
	}
	cr.Spec.SecretRef = corev1.ObjectReference{Name: secretRefName, Namespace: secretRefNamespace}

	return cco.NewServiceID(ccoIBMClient, name, accountID, resourceGroupID, cr), nil
}

// setupServiceID create serviceID and APIKey for credential request yaml passed
func setupServiceID(name, APIKey, accountID, resourceGroupID, crYaml, secretRefName, secretRefNamespace string) (*corev1.Secret, error) {

	serviceID, err := createServiceIDClient(name, APIKey, accountID, resourceGroupID, crYaml, secretRefName, secretRefNamespace)
	if err != nil {
		return nil, fmt.Errorf("error creating serviceID client, err: %w", err)
	}

	if err = serviceID.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate the serviceID %s, err: %w", name, err)
	}

	if err = serviceID.Do(); err != nil {
		return nil, fmt.Errorf("failed to process the serviceID %s, err: %w", name, err)
	}

	secret, err := serviceID.BuildSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to Dump the secret for the serviceID %s, err: %w", name, err)
	}

	return secret, nil
}

// deleteServiceID deletes serviceID and APIkey for credential request yaml passed
func deleteServiceID(name, APIKey, accountID, resourceGroupID, crYaml, secretRefName, secretRefNamespace string) error {
	serviceID, err := createServiceIDClient(name, APIKey, accountID, resourceGroupID, crYaml, secretRefName, secretRefNamespace)
	if err != nil {
		return fmt.Errorf("error creating serviceID client, err: %w", err)
	}

	if err = serviceID.Delete(true); err != nil {
		return err
	}

	return nil
}
