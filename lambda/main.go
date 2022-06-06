package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/google/uuid"
)

type CreateTodoRequest struct {
	Name        string    `json:"name" dynamodbav:"name"`
	Description string    `json:"description" dynamodbav:"description, omitempty"`
	CreateDate  time.Time `json:"createDate" dynamodbav:"createDate"`
}

type ListTodoResponse struct {
	ID          string    `json:"id" dynamodbav:"id"`
	Name        string    `json:"name" dynamodbav:"name"`
	Description string    `json:"description" dynamodbav:"description, omitempty"`
	CreateDate  time.Time `json:"createDate" dynamodbav:"createDate"`
}

type UpdateTodoRequest struct {
	ID          string `json:"id" dynamodbav:"id"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description, omitempty"`
}

type EndpointAction func(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

var actions map[string]EndpointAction

var svc *dynamodb.DynamoDB

var tableName string

func init() {
	session := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc = dynamodb.New(session)

	actions = make(map[string]EndpointAction)
	actions[http.MethodPost] = saveTodo
	actions[http.MethodGet] = getTodos
	actions[http.MethodPut] = updateTodo
	actions[http.MethodDelete] = removeTodo

	tableName = os.Getenv("TableName")
}

func handler(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	action := actions[event.HTTPMethod]
	if action != nil {
		return action(event)
	}

	return httpError(http.StatusInternalServerError, nil)
}

func getTodos(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	input := dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	result, err := svc.ScanWithContext(ctx, &input)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err

	}

	var todos []ListTodoResponse
	for _, v := range result.Items {
		todo := ListTodoResponse{}

		err := dynamodbattribute.UnmarshalMap(v, &todo)
		if err != nil {
			return httpError(http.StatusInternalServerError, err)
		}
		todos = append(todos, todo)
	}

	todoToJson, err := json.Marshal(todos)
	if err != nil {
		return httpError(http.StatusInternalServerError, err)

	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(todoToJson),
	}, nil
}

func saveTodo(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var createTodoRequest CreateTodoRequest
	err := json.Unmarshal([]byte(event.Body), &createTodoRequest)
	if err != nil {
		return httpError(http.StatusBadRequest, err)
	}

	av, err := dynamodbattribute.MarshalMap(createTodoRequest)
	if err != nil {
		return httpError(http.StatusInternalServerError, err)
	}
	av["id"] = &dynamodb.AttributeValue{
		S: aws.String(uuid.NewString()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	input := dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	}

	_, err = svc.PutItemWithContext(ctx, &input)
	if err != nil {
		return httpError(http.StatusInternalServerError, err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
	}, nil
}

func updateTodo(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := event.PathParameters["id"]
	if !ok {
		return httpError(http.StatusBadRequest, nil)
	}

	var updateTodoRequest UpdateTodoRequest
	err := json.Unmarshal([]byte(event.Body), &updateTodoRequest)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if id != updateTodoRequest.ID {
		return httpError(http.StatusBadRequest, nil)

	}

	updateBuilder := expression.
		Set(expression.Name("name"), expression.Value(updateTodoRequest.Name)).
		Set(expression.Name("description"), expression.Value(updateTodoRequest.Description)).
		Set(expression.Name("updateDate"), expression.Value(time.Now().UTC().Format(time.RFC3339)))

	expr, err := expression.NewBuilder().WithUpdate(updateBuilder).Build()
	if err != nil {
		return httpError(http.StatusInternalServerError, err)
	}

	_, err = svc.UpdateItem(&dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		TableName:                 aws.String(tableName),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	if err != nil {
		return httpError(http.StatusInternalServerError, err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}

func removeTodo(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := event.PathParameters["id"]
	if !ok {
		return httpError(http.StatusBadRequest, nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	input := dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
	}

	_, err := svc.DeleteItemWithContext(ctx, &input)
	if err != nil {
		return httpError(http.StatusInternalServerError, err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}

func httpError(statusCode int, err error) (events.APIGatewayProxyResponse, error) {
	if err != nil {
		log.Println(err.Error())
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       http.StatusText(statusCode),
	}, nil
}

func main() {
	lambda.Start(handler)
}
