# Deploying Caduceator

## Docker

In order to deploy into Docker, make sure [Docker is installed](https://docs.docker.com/install/). You will also need to have the AWS CLI installed. If you are on Mac OSX, you can use homebrew to install it by executing this command on the terminal: `brew install awscli`

#### Deploy

1. Clone this repository

2. [Install AWS CLI.](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)

3. Run `deploy/docker-compose/deploy.sh`
   
    This will run `docker-compose up` which uses images of `caduceus`, `argus`, and `caduceator`, from dockerhub. 

    To pull specific versions of the images, just set the `<SERVICE>_VERSION` env variables when running the shell script.

    ```
    CADUCEUS_VERSION=x.x.x deploy/docker-compose/deploy.sh
    ```

    If you only want to bring up, for example, the scytale and talaria, run:
    ```bash
    deploy/docker-compose/deploy.sh scytale talaria
    ```
    _**Note**_: Bringing up a subset of services can cause problems.
    
    This can be done with any combination of services. If you want to use a locally built image of any service, make sure you build a local image first by executing `make local-docker` in the level of the repo where the Makefile lives, and then you can change the version in `deploy.sh`. For example, if you wanted to use a local image of caduceus:

    ```
    CADUCEUS_VERSION=${CADUCEUS_VERSION:-local}
    ```

4. To bring the containers down:
   ```bash
   docker-compose -f deploy/docker-compose/docker-compose.yml down
   ```

### Config Files
You can change the configurations of different services by modifying the yaml files found in the `docFiles/` folder. To apply the config changes, tear down the container and rebuild it. 