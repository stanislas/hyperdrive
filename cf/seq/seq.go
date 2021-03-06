// # Sequence Generator
//
// The `seq` custom resource is used to create a sequence that is stored as an SSM parameter.
// Once created and used, the sequence initial value can not more be changed.
//
// To fetch values from the sequence, use the `seqval` custom resource.
//
// ## Syntax
//
// To create an `seq` resource, add the following resource to your cloudformation
// template (yaml notation, json is similar)
//
// ```yaml
// MySequence:
//   Type: Custom::SequenceGenerator
//   Properties:
//     ServiceToken:
//       Fn::ImportValue:
//         !Sub ${HyperdriveCore}-SequenceGenerator
//     SequenceName: /parameter/name
//     InitialValue: 1
// ```
//
// ## Properties
//
// `SequenceName`
//
// > The name of the sequence that will be a suffix for the underlying SSM parameter. Must start with "/".
// > The parameters name have the prefix "/hyperdrive/sequence".
//
// _Type_: String
//
// _Required_: Yes
//
// _Update Requires_: replacement
//
// `Expression`
//
// > The arithmetic expression to compute the sequence: standard operations + variable x for the current value of the
// > sequence.
// > WARNING: changing the expression is dangerous as the sequence could retake previously issued values.
// >
// > _Type_: String
// >
// > _Default_: x
// >
// > _Required_: No
// >
// > _Update Requires_: no interruption
//
// ## Return Values
//
// `Ref`
//
// The `Ref` intrinsic function gives the name of the created SSM parameter
package main

import (
	"context"
	"github.com/DEEP-IMPACT-AG/hyperdrive/common"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	awsssm "github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"strings"
)

var ssm *awsssm.SSM

func main() {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	ssm = awsssm.New(cfg)
	lambda.Start(cfn.LambdaWrap(processEvent))
}

// The SequenceProperties is the main data structure for the resource and
// is defined as a go struct. The struct mirrors the properties as defined above.
// We use the library [mapstructure](https://github.com/mitchellh/mapstructure) to
// decode the generic map from the cloudformation event to the struct.
type SequenceProperties struct {
	SequenceName, Expression string
}

func sequenceProperties(input map[string]interface{}) (SequenceProperties, error) {
	var properties SequenceProperties
	if err := mapstructure.Decode(input, &properties); err != nil {
		return properties, err
	}
	if !strings.HasPrefix(properties.SequenceName, "/") {
		return properties, errors.Errorf("name %s must start with an /", properties.SequenceName)
	}
	if properties.Expression == "" {
		properties.Expression = "x"
	}
	if _, err := common.Eval(properties.Expression, 1); err != nil {
		return properties, err
	}
	return properties, nil
}

func processEvent(ctx context.Context, event cfn.Event) (string, map[string]interface{}, error) {
	properties, err := sequenceProperties(event.ResourceProperties)
	if err != nil {
		return "", nil, err
	}
	switch event.RequestType {
	case cfn.RequestDelete:
		if !common.IsFailurePhysicalResourceId(event.PhysicalResourceID) {
			_, err := ssm.DeleteParameterRequest(&awsssm.DeleteParameterInput{
				Name: &event.PhysicalResourceID,
			}).Send();
			if err != nil {
				return event.PhysicalResourceID, nil, errors.Wrapf(err, "could not delete the sequence %s", properties.SequenceName)
			}
		}
		return event.PhysicalResourceID, nil, nil
	case cfn.RequestCreate:
		return putSequence(ssm, properties)
	case cfn.RequestUpdate:
		return putSequence(ssm, properties);
	default:
		return "", nil, errors.Errorf("unknown request type %s", event.RequestType)
	}
}

func putSequence(ssm *awsssm.SSM, properties SequenceProperties) (string, map[string]interface{}, error) {
	overwrite := true
	parameterName := "/hyperdrive/sequence" + properties.SequenceName
	_, err := ssm.PutParameterRequest(&awsssm.PutParameterInput{
		Name:      &parameterName,
		Type:      awsssm.ParameterTypeString,
		Value:     &properties.Expression,
		Overwrite: &overwrite,
	}).Send();
	if err != nil {
		return "", nil, errors.Wrapf(err, "could not put the parameter %s", parameterName)
	}
	return parameterName, nil, nil
}
