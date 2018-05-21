# build-waiter

`build-waiter` is a utility which will block until all of the previously run builds for a given branch are completed.

For example, this is a sample output of `build-waiter` if we were waiting for a build ahead of us.

```
Waiting on build 8f1076e1-3968-43ea-a366-1c97c1cad27d
Waiting on build 8f1076e1-3968-43ea-a366-1c97c1cad27d
Waiting on build 8f1076e1-3968-43ea-a366-1c97c1cad27d
Waiting on build 8f1076e1-3968-43ea-a366-1c97c1cad27d
Waiting on build 8f1076e1-3968-43ea-a366-1c97c1cad27d
Waiting on build 8f1076e1-3968-43ea-a366-1c97c1cad27d
Resuming build
```

## Usage

`build-waiter` expects the following environment variables to be set:

| Environment Variable    | Description                                    |
| --------------------    | ---------------------------------------------  |
| `CODESHIP_USERNAME`     | Email address of account used for API calls.   |
| `CODESHIP_PASSWORD`     | Password of account used for API calls.        |
| `CODESHIP_ORGANIZATION` | Codeship organization the project resides in.  |
| `CI_PROJECT_ID`         | The UUID of the project for the running build. |
| `CI_BUILD_ID`           | The UUID of build running build-waiter.        |

Note: A dockerized version is available in the [codeship/build-waiter-image repo](https://github.com/codeship/build-waiter-image).

## Development

This project uses [dep](https://github.com/golang/dep) for dependency management.

To install/update dep and all dependencies, run:

```bash
make setup
make dep
```

### Testing

```bash
make test
```
