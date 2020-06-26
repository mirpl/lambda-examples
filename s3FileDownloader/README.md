# Github Actions Envs:
- AWS_ACCESS_KEY_ID - AWS Access key
- AWS_SECRET_ACCESS_KEY - AWS Secret access key

## Lambda Envs:
- S3_ENDPOINT - object storage endpoint, e.g. _s3.amazonaws.com_
- S3_ACCESSKEY - access key for the object storage
- S3_SECRETKEY - secret key for the object storage
- S3_LOCATION - region code, e.g. _us-east-1_
- S3_BUCKET - bucket name


## Test Deployment

#### Deploy with Serverless:

You need to have Serverless installed on your machine.
Once Serverless' installed, go to this README's location in your terminal and type the following command:

```
sls invoke local -f s3FileDownloader \
-e S3_ENDPOINT=<object storage endpoint> \
-e S3_ACCESSKEY=<object storage access key> \
-e S3_SECRETKEY=<object storage secret key ID> \
-e S3_LOCATION=us-east-1 \
-e S3_BUCKET=lambda-example-file-storage \
--path data.json
```
