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
sam build
sam local start-api
```

## Possible Errors while running the above command

- If you don't have docker installed on your machine or if your docker daemon is not running in the background -> `Error: Running AWS SAM projects locally requires Docker. Have you got it installed and running?`

-> [Install docker on your machine](https://docs.docker.com/engine/install/)
-> Run `sudo service docker start` command in the terminal to start docker services on your machine after the installation
-> Run `docker ps` to check if the docker services have been started. This command lists all the running docker containers & it works only when docker [daemon](https://en.wikipedia.org/wiki/Daemon_(computing)#:~:text=In%20multitasking%20computer%20operating%20systems,control%20of%20an%20interactive%20user.&text=Daemons%20such%20as%20cron%20may%20also%20perform%20defined%20tasks%20at%20scheduled%20times.) is running.
-> Run `sudo chmod 666 /var/run/docker.sock` on linux if you are not able to start the docker services. [More about Unix file permissions](https://docs.oracle.com/cd/E19253-01/816-4557/secfile-60/index.html#:~:text=A%20text%20file%20has%20666,%2Fetc%2Fprofile%20file%2C%20.)
* Re-run `sudo service docker start` to start docker services

* Now re-run `sam local start-api` again

You can see service running on localhost:3000
