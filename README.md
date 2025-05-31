# Identity Service

![Sequence Diagram](https://user-images.githubusercontent.com/45519620/176491640-6f58d7d5-6fe1-42d9-a9d6-fc1b23e0dee2.jpg)

The backend service serves as the interface between multiple user services and main nodejs backend. The purpose of the service is to provide a centralized platform for storing and accessing profile data of users. The hosted service is responsible for handling the interaction between the user services and the main nodejs backend. The user is responsible for developing, deploying, maintaining, and enhancing a service named `Profile Service` to ensure that it continues to meet the evolving needs of the website. The profile data stored on the service is primarily used on the members page of the realdevsquad.com website, providing a comprehensive and up-to-date view of the community.

## Table of Contents
- [Installation](#installation)
- [Run](#run)
- [Usage](#usage)
- [Features](#features)
- [Contributing](#contributing)
- [FAQs](#faqs)

## Installation
You should have some things pre-installed :
- [VSCode](https://code.visualstudio.com/)
- [Git](https://git-scm.com/)
- [SAM-CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- [GOLANG](https://go.dev/)
- [DOCKER](https://www.docker.com/)

**Open the terminal or command prompt:** Depending on your operating system, open the terminal or command prompt to begin the cloning process.

**Navigate to your desired local directory:** Use the cd command to navigate to the directory where you want to store the cloned repository.

**Clone the repository:** Use the following command to clone the repository :

```
git clone https://github.com/Real-Dev-Squad/identity-service.git
```

# Setting Up the Environment for Your Local Server

## Firestore Setup

1. Create an application on [Firebase Firestore](https://firebase.google.com/docs/firestore)
2. [Generate a service account key file](https://cloud.google.com/iam/docs/creating-managing-service-account-keys)

## Environment Configuration

1. Create an `env.json` file by copying the `sample-env.json` template
2. Add your Firestore service credentials to the file

### Formatting Service Credentials
When adding the service account JSON to your `env.json` file:
- Convert the entire JSON to a single line string
- Escape special characters:
  - Replace all newlines (`\n`) with escaped newlines (`\\n`)
  - Replace all double quotes (`"`) with escaped double quotes (`\"`)

### Example
If your service account JSON looks like this:
```json
{
  "type": "service_account",
  "project_id": "your-project-id",
  "private_key_id": "key-id",
  "private_key": "-----BEGIN PRIVATE KEY-----\nkey-content\n-----END PRIVATE KEY-----\n",
  "client_email": "email@example.com"
}
```

It should be added to `env.json` as:
```json
{
  "FIRESTORE_CREDENTIALS": "{\"type\":\"service_account\",\"project_id\":\"your-project-id\",\"private_key_id\":\"key-id\",\"private_key\":\"-----BEGIN PRIVATE KEY-----\\nkey-content\\n-----END PRIVATE KEY-----\\n\",\"client_email\":\"email@example.com\"}"
}
```

## 3. Running the Server

After completing the above steps, you can now run the server locally with your configured environment.

### Executing the script to run the server locally

- Windows users need to download & install [Git bash](https://gitforwindows.org/) to execute the script.
- Mac/Linux users can run the script in their native terminal.

```
sh scripts/dev.sh
```

[Possible Errors while running the above command](DOCKERERRORS.md)

**You can see service running on localhost:3000.**
## Usage
- **User services send profile data to the Identity service:** The user services, which are developed, deployed, and maintained by the user, send profile data to the Identity service. This profile data is stored in a centralized database.

- **Identity service stores profile data:** The Identity service receives the profile data from the user services and stores it in a centralized database. This ensures that all the profile data is in one place, making it easier to access and manage.

- User services can update profile data: The user services can also update profile data stored in the centralized database through the Identity service. This ensures that the profile data is up-to-date and reflects any changes made by the user services.

## Features
We are using AWS SAM(Serverless Application Model) with [golang](https://go.dev/). The AWS Serverless Application Model (SAM) is an open-source framework for building serverless applications. It provides shorthand syntax to express functions, APIs, databases, and event source mappings. With just a few lines per resource, you can define the application you want and model it using YAML. Read more about SAM [here](https://aws.amazon.com/serverless/sam/).

### *Routes Created*

```
/profile
/verify
/health
```

## State Machine Diagram
![statemachinediagram drawio](https://user-images.githubusercontent.com/45519620/228067673-914ee0c9-9980-48d9-a012-8c30872671ea.png)

## Contributing
Wish to contribute? You can find a detailed guide [here](./CONTRIBUTING.md)

## FAQs
For FAQs, you can check this [discussion](https://github.com/Real-Dev-Squad/identity-service/discussions/102).
