# Deploying Stratum to Google Cloud: A Comprehensive Guide

Welcome to the comprehensive guide for deploying your Stratum instance to Google Cloud. This document will walk you through every step of the process, from an initial test deployment to a secure, scalable, production-ready architecture.

---

## **Part 1: Your First Deployment**

In this section, we'll get your application running on Cloud Run as quickly as possible. This is a great way to verify that your container works in the cloud environment before adding production-level components.

### 1.1. Prerequisites

Before we begin, please ensure you have the following:

-   **Google Cloud Project**: A project with active billing. If you don't have one, you can create one from the [Google Cloud Console](https://console.cloud.google.com/).
-   **`gcloud` CLI**: The Google Cloud SDK must be installed and authenticated on your local machine. For help, follow the [official installation instructions](https://cloud.google.com/sdk/docs/install).

### 1.2. Project Configuration

First, let's set up some environment variables to make the following commands easier to manage. Be sure to replace `[PROJECT_ID]` with your actual Google Cloud project ID.

```bash
# Replace [PROJECT_ID] with your actual Google Cloud Project ID
export PROJECT_ID=[PROJECT_ID]

# Choose a region for your services (e.g., us-central1)
export REGION=us-central1

# Set the gcloud CLI to use your configured project and region
gcloud config set project $PROJECT_ID
gcloud config set run/region $REGION
```

### 1.3. Enable Required APIs

Next, we need to enable the necessary Google Cloud services for our project. This command ensures that Cloud Build, Artifact Registry, and Cloud Run are ready to use.

```bash
gcloud services enable run.googleapis.com artifactregistry.googleapis.com cloudbuild.googleapis.com
```

### 1.4. Create an Artifact Registry Repository

Artifact Registry is Google Cloud's recommended service for storing and managing container images. We'll create a dedicated repository to hold our Stratum application image.

```bash
gcloud artifacts repositories create stratum \
    --repository-format=docker \
    --location=$REGION \
    --description="Stratum application container repository"
```

### 1.5. Build and Push the Container Image

With the repository in place, we can now use Cloud Build to containerize our application. This command finds the `Dockerfile` in your project, builds the image, and pushes it to the Artifact Registry repository we just created.

```bash
gcloud builds submit --tag $REGION-docker.pkg.dev/$PROJECT_ID/stratum/stratum:latest
```

### 1.6. Deploy to Cloud Run

Now for the exciting part: deploying the application. The following command will create a new Cloud Run service and deploy your container to it, making it accessible via a public URL.

```bash
gcloud run deploy stratum \
    --image $REGION-docker.pkg.dev/$PROJECT_ID/stratum/stratum:latest \
    --platform managed \
    --allow-unauthenticated
```
-   The `--allow-unauthenticated` flag makes the service publicly accessible.

At this point, your service is live! However, it is not yet configured with the necessary environment variables to function correctly. The next section will guide you through creating a more robust, production-ready setup.

---

## **Part 2: Production-Ready Architecture**

A production environment requires greater security and scalability. In this section, we will enhance our deployment by using managed services for dependencies like Redis and connecting them securely over a private network.

### 2.1. Enable Additional APIs

First, we need to enable the APIs for Serverless VPC Access and Memorystore for Redis.

```bash
gcloud services enable vpcaccess.googleapis.com redis.googleapis.com
```

### 2.2. Set Up a VPC Network and Connector

To allow your Cloud Run service to communicate privately with other Google Cloud services, we need to set up a Virtual Private Cloud (VPC) network and a Serverless VPC Access connector.

```bash
# Create a VPC network for your services
gcloud compute networks create stratum-vpc --subnet-mode=auto

# Create a Serverless VPC Access connector within the network
gcloud compute networks vpc-access connectors create stratum-connector \
    --network stratum-vpc \
    --region=$REGION \
    --range "10.8.0.0/28"
```

### 2.3. Provision a Memorystore for Redis Instance

Next, we'll create a fully managed Redis instance using Google Cloud Memorystore. This will serve as our application's cache.

```bash
gcloud redis instances create stratum-redis \
    --size=1 \
    --region=$REGION \
    --tier=BASIC \
    --redis-version=REDIS_6_X \
    --network=stratum-vpc
```
-   This command may take a few minutes to complete.
-   The `BASIC` tier is suitable for development and small applications. For production systems requiring high availability, consider the `STANDARD_HA` tier.

Once the instance is ready, retrieve its private IP address and store it in an environment variable.
```bash
export REDIS_HOST=$(gcloud redis instances describe stratum-redis --region=$REGION --format "value(host)")
```

### 2.4. Deploy with Production Configuration

Finally, let's re-deploy our Cloud Run service with its full production configuration. This command connects the service to our VPC and securely passes the Redis connection URL and other project-specific settings as environment variables.

```bash
gcloud run deploy stratum \
    --image $REGION-docker.pkg.dev/$PROJECT_ID/stratum/stratum:latest \
    --platform managed \
    --allow-unauthenticated \
    --vpc-connector stratum-connector \
    --set-env-vars "REDIS_URL=redis://$REDIS_HOST:6379" \
    --set-env-vars "^,^PROJECT_1_ROUTE=/example/{id},PROJECT_1_DB_DSN=..."
```
-   `--vpc-connector`: Attaches the service to your private VPC network.
-   `--set-env-vars`: Use this flag to set your `REDIS_URL` and all `PROJECT_*` variables required by Stratum. Use the `^,^` delimiter to set multiple variables in a single command.

Your Stratum instance is now fully deployed and configured for production use.

---

## **Part 3: Cleaning Up**

To avoid incurring future charges from the resources you've created, it's important to clean up your project. The following commands will delete the services and infrastructure we set up in this guide.

```bash
# Delete the Cloud Run service
gcloud run services delete stratum --quiet

# Delete the Memorystore instance
gcloud redis instances delete stratum-redis --region=$REGION --quiet

# Delete the VPC Access connector
gcloud compute networks vpc-access connectors delete stratum-connector --region=$REGION --quiet

# Delete the VPC network
gcloud compute networks delete stratum-vpc --quiet

# Delete the Artifact Registry repository
gcloud artifacts repositories delete stratum --location=$REGION --quiet
```
