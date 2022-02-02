# identity-service

The part of the website which holds the identity of members

# How can I contribute?

Wish to contribute? You can find a detailed guide [here](./CONTRIBUTING.md)!

## Project Structure

We are using AWS SAM(Serverless Application Model) with [golang](https://go.dev/). The AWS Serverless Application Model (SAM) is an open-source framework for building serverless applications. It provides shorthand syntax to express functions, APIs, databases, and event source mappings. With just a few lines per resource, you can define the application you want and model it using YAML. Read more about SAM [here](https://aws.amazon.com/serverless/sam/).

#### Routes Created

```
/getData
/verify
/health-check
```

## How to start ?

You should have some things pre-installed -

[SAM-CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
[GOLANG](https://go.dev/)

## To Run locally

### Firestore setup before running the server locally

- Create an application on [FireStore](https://firebase.google.com/docs/firestore) and [generate a service file](https://cloud.google.com/iam/docs/creating-managing-service-account-keys).
- Add the service file credentials in the sample-env.json file as a string.
- Remove all the spaces such that the whole _json_ that you copied is in a single line
- Replace **\n** with **\\\\n** in your copied json
- Replace **"** with **\\"** in your copied json

### Executing the script to run the server locally

- Windows users need to download & install [Git bash](https://gitforwindows.org/) to execute the scirpt.
- Mac/Linux users can run the script in your native terminal.

```
sh scripts/dev.sh
```

[Possible Errors while running the above command](DOCKERERRORS.md)

You can see service running on localhost:3000
