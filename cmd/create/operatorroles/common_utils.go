package operatorroles

import (
	"fmt"

	"github.com/openshift-online/ocm-common/pkg"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
)

const assumePolicyAction = "sts:AssumeRole"

func computePolicyARN(accountID string, prefix string, namespace string, name string, path string) string {
	if prefix == "" {
		prefix = aws.DefaultPrefix
	}
	policy := fmt.Sprintf("%s-%s-%s", prefix, namespace, name)
	if len(policy) > pkg.MaxByteSize {
		policy = policy[0:pkg.MaxByteSize]
	}
	if path != "" {
		return fmt.Sprintf("arn:%s:iam::%s:policy%s%s", aws.GetPartition(), accountID, path, policy)
	}
	return fmt.Sprintf("arn:%s:iam::%s:policy/%s", aws.GetPartition(), accountID, policy)
}

func validateIngressOperatorPolicyOverride(r *rosa.Runtime, policyArn string, sharedVpcRoleArn string,
	installerRolePrefix string) error {
	_, err := r.AWSClient.IsPolicyExists(policyArn)
	policyExists := err == nil
	if !policyExists {
		return nil
	}

	policyDocument, err := r.AWSClient.GetDefaultPolicyDocument(policyArn)
	if err != nil {
		return err
	}

	document, err := aws.ParsePolicyDocument(policyDocument)
	if err != nil {
		return err
	}

	for _, statement := range document.Statement {
		if statement.Action == assumePolicyAction && statement.Effect == "Allow" {
			// The policy associated with the installer role. In the case it contains a different shared VPC role ARN,
			// don't override it.
			if statement.Resource != sharedVpcRoleArn {
				return errors.UserErrorf("Policy with ARN '%s' contains '%s' with an unexpected shared VPC role ARN "+
					"[Expected: '%s', Provided '%s'].\n"+
					"The policy is associated with the installer role with the prefix '%s'.\n"+
					"To create operator roles with shared VPC role ARN '%s', please provide a different value for "+
					"'--installer-role-arn'", policyArn, assumePolicyAction, statement.Resource, sharedVpcRoleArn,
					installerRolePrefix, sharedVpcRoleArn)
			}
		}
	}

	return nil
}
