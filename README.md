# CORGI 🐕

Cilium OpenseaRch GitHub Ingester is a connector for ingesting Cilium CI
data into OpenSearch.

## Workflow Runs

Use the `workflow runs` sub-command to download workflow runs for the target repository.
A bulk request is printed on stdout that can be given to OpenSearch for indexing.

Each document contains the following:

* Information regarding the workflow run.
* Jobs contained in the workflow
* Steps contained in the workflow
* Tests contained in the workflow, if a `cilium-junits` artifact is present.

This outputted bulk request may be too large to send to OpenSearch in onen go, therefore one can leverage the `split` command to break the request up into smaller chunks.

Example usage:


```shell
docker-compose up -d
go run . workflow runs > out.json
split -l 250 out.json split
find . -name "split*" -exec \   
    curl -XPUT --data-binary @{} --insecure -H "Content-Type: application/json" -u \
        admin:$OPENSEARCH_INITIAL_ADMIN_PASSWORD https://localhost:9200/_bulk --verbose \;
```
