package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/jsii-runtime-go"

	// "github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	// "github.com/aws/jsii-runtime-go"
)

const (
	tableName = "todo-api"
)

type CdkTodoApiStackProps struct {
	awscdk.StackProps
}

func NewCdkTodoApiStack(scope constructs.Construct, id string, props *CdkTodoApiStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// The code that defines your stack goes here
	table := awsdynamodb.NewTable(stack, jsii.String(tableName), &awsdynamodb.TableProps{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("id"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TableName: jsii.String(tableName),
	})

	myLambda := configureLambdaStack(stack, table)

	configureApiGatewayStack(stack, myLambda)

	return stack
}

func configureLambdaStack(stack awscdk.Stack, table awsdynamodb.Table) awslambda.Function {
	myLambda := awslambda.NewFunction(stack, jsii.String("TodoFunction"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_GO_1_X(),
		Handler: jsii.String("lambdaHandler"),
		Code:    awslambda.AssetCode_FromAsset(jsii.String("./lambda"), &awss3assets.AssetOptions{}),
		Environment: &map[string]*string{
			"TableName": jsii.String(tableName),
		},
	})

	table.GrantFullAccess(myLambda)

	return myLambda
}

func configureApiGatewayStack(stack awscdk.Stack, myLambda awslambda.Function) {
	api := awsapigateway.NewRestApi(stack, jsii.String("todo-apigw"), &awsapigateway.RestApiProps{
		RestApiName: jsii.String("Todo Lambda Service"),
		Description: jsii.String("This service for demonstration"),
	})

	target := awsapigateway.NewLambdaIntegration(myLambda, &awsapigateway.LambdaIntegrationOptions{
		RequestTemplates: &map[string]*string{
			"application/json": jsii.String("{ 'statusCode': '200' }"),
		},
	})

	resource := api.Root().AddResource(jsii.String("todos"), &awsapigateway.ResourceOptions{})
	resource.AddMethod(jsii.String("POST"), target, &awsapigateway.MethodOptions{})
	resource.AddMethod(jsii.String("GET"), target, &awsapigateway.MethodOptions{})

	todo := resource.AddResource(jsii.String("{id}"), &awsapigateway.ResourceOptions{})
	todo.AddMethod(jsii.String("PUT"), target, &awsapigateway.MethodOptions{})
	todo.AddMethod(jsii.String("DELETE"), target, &awsapigateway.MethodOptions{})
}

func main() {
	app := awscdk.NewApp(nil)

	NewCdkTodoApiStack(app, "CdkTodoApiStack", &CdkTodoApiStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return &awscdk.Environment{
		Region: jsii.String("eu-central-1"),
	}

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}
