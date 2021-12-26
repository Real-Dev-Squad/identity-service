# identity-service

The part of the website which holds the identity of members

# How can I contribute?

Wish to contribute? You can find a detailed guide [here](./CONTRIBUTING.md)!

## Project Structure

We are using AWS SAM(Serverless Application Model) with [golang](https://go.dev/). The AWS Serverless Application Model (SAM) is an open-source framework for building serverless applications. It provides shorthand syntax to express functions, APIs, databases, and event source mappings. With just a few lines per resource, you can define the application you want and model it using YAML. Read more about SAM [here](https://aws.amazon.com/serverless/sam/).

#### Routes Created

```
/health
/verify
```

## How to start ?

You should have some things pre-installed -

[SAM-CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
[GOLANG](https://go.dev/)

## To Run locally

```
sam local start-api
```

You can see service running on localhost:3000