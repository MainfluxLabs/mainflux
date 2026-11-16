# Pyscripts Service

The Pyscripts service stores Python scripts, runs them on demand against a message, and records the outcome of every run.

## Scripts

| Field         | Description                                    |
|---------------|------------------------------------------------|
| `id`          | Unique script identifier (UUID)                |
| `group_id`    | ID of the group the script belongs to          |
| `name`        | Human-readable script name                     |
| `description` | Optional description                           |
| `source`      | Python source code                             |

Scripts are group-scoped and authorized through the Things service: group viewers can read scripts and runs, group editors can create, update, remove and run them. Running is an editor action because it executes code inside the service.

Source is checked when it is saved: `POST /groups/:id/scripts` and `PUT /scripts/:id` reject a script that is larger than `MF_PYSCRIPTS_MAX_SCRIPT_SIZE` or does not compile with `400 Bad Request`. One create request accepts at most 32 scripts.

When a group is removed, its scripts and their runs are removed with it.

## Writing a script

A script is plain Python that runs top to bottom. The input is available as the global `message`:

```python
readings = [r["v"] for r in message["payload"]["readings"]]
print("saw", len(readings), "readings")
sum(readings) / len(readings)
```

- `message["payload"]` is always present. `subtopic`, `created` and `thing_id` are present when the caller supplies them.
- If the last statement is an expression, its value is the run's result. Otherwise the result is `null`.
- The result must be JSON-serializable and at most 4 MiB. NumPy scalars, arrays, DataFrames, `NaN` and `Infinity` fail the run with `result cannot be returned: ...`; convert them with `.tolist()` or `float()` first.
- `print()` and writes to `sys.stderr` are recorded as the run's logs, up to `MF_PYSCRIPTS_MAX_LOG_LINES` lines.
- An exception fails the run, and its traceback is recorded as the run's error.
- `numpy` and `pandas` are available. No other packages can be installed.

## Runs

`POST /scripts/:id/run` takes a JSON body with `payload` (at most 1 MiB) and optional `subtopic`, `created` and `thing_id`, executes the script, and stores the run.

A script that raises still returns `200 OK`, with `status: "fail"` and the traceback in `error`. Other status codes:

| Status | Cause                                                                  |
|--------|------------------------------------------------------------------------|
| `429`  | The group exceeded its rate limit (2 requests/s, burst of 10)          |
| `503`  | No worker became free within `MF_PYSCRIPTS_ACQUIRE_TIMEOUT`            |
| `500`  | The runner failed for a reason outside the script                      |

The rate limit is per group and applies to creating, updating and running scripts.

`GET /scripts/:id/runs` lists a script's runs, newest first by default (`order=started_at`, `dir=desc`), and can be filtered by `status` (`success`, `fail`) and a `from`/`to` range in Unix milliseconds. Run history is kept until it is removed with `PATCH /runs`.

## Sandboxing

At startup the service launches `MF_PYSCRIPTS_WORKERS` Python workers that import NumPy and pandas once. Each run is executed in a child forked from a worker, so no run can affect the next one. Before any user code runs, the child:

- sets `RLIMIT_CPU`, `RLIMIT_DATA`, `RLIMIT_FSIZE` and disables core dumps,
- installs a seccomp allow-list filter that denies networking, new processes, file writes and signals to other processes,
- starts a new session and replaces stdin, stdout and stderr with `/dev/null`.

The service kills the run's process group when `MF_PYSCRIPTS_RUN_TIMEOUT` expires and refuses to start a worker that reports no seccomp filter, so `pyseccomp` and `libseccomp` must be installed in the image.

This is meant for scripts written by trusted group editors, not for hostile multi-tenant code. Script source and run logs are readable by every group viewer, so they must not contain credentials.

## Configuration

The service is configured using the environment variables from the following table. Note that any unset variables will be replaced with their default values.

| Variable                         | Description                                                                | Default                  |
|----------------------------------|----------------------------------------------------------------------------|--------------------------|
| `MF_PYSCRIPTS_LOG_LEVEL`         | Log level for Pyscripts (debug, info, warn, error)                         | error                    |
| `MF_PYSCRIPTS_DB_HOST`           | Database host address                                                      | localhost                |
| `MF_PYSCRIPTS_DB_PORT`           | Database host port                                                         | 5432                     |
| `MF_PYSCRIPTS_DB_USER`           | Database user                                                              | mainflux                 |
| `MF_PYSCRIPTS_DB_PASS`           | Database password                                                          | mainflux                 |
| `MF_PYSCRIPTS_DB`                | Name of the database used by the service                                   | pyscripts                |
| `MF_PYSCRIPTS_DB_SSL_MODE`       | Database connection SSL mode (disable, require, verify-ca, verify-full)    | disable                  |
| `MF_PYSCRIPTS_DB_SSL_CERT`       | Path to the PEM encoded certificate file                                   |                          |
| `MF_PYSCRIPTS_DB_SSL_KEY`        | Path to the PEM encoded key file                                           |                          |
| `MF_PYSCRIPTS_DB_SSL_ROOT_CERT`  | Path to the PEM encoded root certificate file                              |                          |
| `MF_PYSCRIPTS_CLIENT_TLS`        | Flag that indicates if TLS should be turned on                             | false                    |
| `MF_PYSCRIPTS_CA_CERTS`          | Path to trusted CAs in PEM format                                          |                          |
| `MF_PYSCRIPTS_HTTP_PORT`         | Pyscripts service HTTP port                                                | 9032                     |
| `MF_PYSCRIPTS_SERVER_CERT`       | Path to server certificate in pem format                                   |                          |
| `MF_PYSCRIPTS_SERVER_KEY`        | Path to server key in pem format                                           |                          |
| `MF_JAEGER_URL`                  | Jaeger server URL for distributed tracing. Leave empty to disable tracing. |                          |
| `MF_THINGS_AUTH_GRPC_URL`        | Things auth service gRPC URL                                               | localhost:8183           |
| `MF_THINGS_AUTH_GRPC_TIMEOUT`    | Things auth service gRPC request timeout                                   | 1s                       |
| `MF_AUTH_GRPC_URL`               | Auth service gRPC URL                                                      | localhost:8181           |
| `MF_AUTH_GRPC_TIMEOUT`           | Auth service gRPC request timeout                                          | 1s                       |
| `MF_PYSCRIPTS_ES_URL`            | Event store URL                                                            | redis://localhost:6379/0 |
| `MF_PYSCRIPTS_PYTHON`            | Python interpreter used for the workers                                    | python3                  |
| `MF_PYSCRIPTS_WORKERS`           | Number of pre-warmed workers, i.e. the number of concurrent runs           | 2                        |
| `MF_PYSCRIPTS_RUN_TIMEOUT`       | Wall-clock limit of one run                                                | 10s                      |
| `MF_PYSCRIPTS_ACQUIRE_TIMEOUT`   | How long a request waits for a free worker before failing with 503         | 5s                       |
| `MF_PYSCRIPTS_START_TIMEOUT`     | How long a worker may take to start                                        | 60s                      |
| `MF_PYSCRIPTS_MAX_SCRIPT_SIZE`   | Maximum script source size in bytes                                        | 65535                    |
| `MF_PYSCRIPTS_MAX_OUTPUT_SIZE`   | Maximum output a run may produce, in bytes                                 | 1048576                  |
| `MF_PYSCRIPTS_MAX_LOG_LINES`     | Maximum number of log lines kept per run                                   | 256                      |
| `MF_PYSCRIPTS_MAX_CPU_SECONDS`   | CPU time limit of one run (`RLIMIT_CPU`), in seconds                       | 5                        |
| `MF_PYSCRIPTS_MAX_MEMORY`        | Memory limit of one run (`RLIMIT_DATA`), in bytes                          | 268435456                |
| `MF_PYSCRIPTS_MAX_FILE_SIZE`     | File size limit of one run (`RLIMIT_FSIZE`), in bytes                      | 0                        |
| `MF_PYSCRIPTS_BLAS_THREADS`      | Number of BLAS/OpenMP threads per run                                      | 1                        |

In docker-compose the container is also bounded by `MF_PYSCRIPTS_MEM_LIMIT` and `MF_PYSCRIPTS_PIDS_LIMIT`. Keep `MF_PYSCRIPTS_MEM_LIMIT` above `MF_PYSCRIPTS_WORKERS × (~120 MB + MF_PYSCRIPTS_MAX_MEMORY)` and below the memory available to Docker, and scale the three together.

## Deployment

The service is distributed as a Docker container built from `docker/Dockerfile.python` (`make docker_pyscripts`). Check the [`pyscripts`](https://github.com/MainfluxLabs/mainflux/blob/master/docker/docker-compose.yml) service section in docker-compose to see how the service is deployed.

To start the service outside of the container, Python 3 with `numpy`, `pandas` and `pyseccomp` (and `libseccomp`) must be installed on a Linux host:

```bash
# Download the latest version of the service
git clone https://github.com/MainfluxLabs/mainflux

cd mainflux

# compile the pyscripts service
make pyscripts

# Copy binary to bin
make install

# set the environment variables and run the service
MF_PYSCRIPTS_LOG_LEVEL=[Pyscripts log level] \
MF_PYSCRIPTS_DB_HOST=[Database host address] \
MF_PYSCRIPTS_DB_PORT=[Database host port] \
MF_PYSCRIPTS_DB_USER=[Database user] \
MF_PYSCRIPTS_DB_PASS=[Database password] \
MF_PYSCRIPTS_DB=[Name of the database used by the service] \
MF_PYSCRIPTS_HTTP_PORT=[Service HTTP port] \
MF_THINGS_AUTH_GRPC_URL=[Things auth service gRPC URL] \
MF_AUTH_GRPC_URL=[Auth service gRPC URL] \
MF_PYSCRIPTS_ES_URL=[Event store URL] \
MF_PYSCRIPTS_PYTHON=[Python interpreter] \
$GOBIN/mainfluxlabs-pyscripts
```

## Usage

For the full HTTP API reference, see the [OpenAPI specification](https://mainfluxlabs.github.io/docs/swagger/).
