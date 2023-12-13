import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";
import * as awsx from "@pulumi/awsx";


const coreInfra = new pulumi.StackReference("honeycomb-devrel/booth-game/booth-game");

const apigatewayId = coreInfra.requireOutput("apiGatewayId");

const gateway = apigatewayId.apply(id => aws.apigatewayv2.getApi({apiId: id}));
const functionName = ""

const lambdaLoggingPolicyDocument = aws.iam.getPolicyDocument({
    statements: [{
        effect: "Allow",
        actions: [
            "logs:CreateLogGroup",
            "logs:CreateLogStream",
            "logs:PutLogEvents",
        ],
        resources: ["arn:aws:logs:*:*:*"],
    }],
});

// functionexecution role for lambda
const role = new aws.iam.Role("execution-role", {
    assumeRolePolicy: aws.iam.assumeRolePolicyForPrincipal({ Service: "lambda.amazonaws.com" }),
    inlinePolicies: [
        {
            name: "executionPolicy",
            policy: lambdaLoggingPolicyDocument.then(policy => policy.json),
        }
    ]});
    
    
const myLambda = new aws.lambda.Function("api-lambda", {
    role: role.arn,
    runtime: aws.lambda.Go1dxRuntime,

    code: new pulumi.asset.FileArchive("../HandleRequest.zip"),
    handler: "HandleRequest",
});

const integration = new aws.apigatewayv2.Integration("api-gateway-integration", {
    apiId: gateway.id,
    integrationType: "AWS_PROXY",
    integrationUri: myLambda.arn,
    payloadFormatVersion: "2.0",
});

// attach integration to route
const route = new aws.apigatewayv2.Route("api-gateway-route", {
    apiId: gateway.id,
    routeKey: "$default",
    target: pulumi.interpolate`integrations/${integration.id}`,
});

// api gateway stage
const stage = new aws.apigatewayv2.Stage("api-gateway-stage", {
    apiId: gateway.id,
    name: "$default",
    autoDeploy: true,
});

var lambdaPermission = new aws.lambda.Permission("api-lambda-permission", {
    action: "lambda:InvokeFunction",
    function: myLambda.name,
    principal: "apigateway.amazonaws.com",
    sourceArn: pulumi.interpolate`${gateway.executionArn}/*/*`,
});
