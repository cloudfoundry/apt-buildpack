# Cloud Foundry Experimental Apt Buildpack

[![CF Slack](https://www.google.com/s2/favicons?domain=www.slack.com) Join us on Slack](https://cloudfoundry.slack.com/messages/buildpacks/)

A Cloud Foundry [buildpack](http://docs.cloudfoundry.org/buildpacks/) for apps requiring custom apt packages.


### Buildpack User Documentation

The apt buildpack can be used to install deb packages prior to use in another buildpack. To configure which packages to install, provide `apt.yml` in your application and include a list of packages to install, eg:

```
---
packages:
- ascii
- libxml
- https://example.com/exciting.deb
```

If you would like to use custom apt repositories, you can add `keys` and `repos` to the `apt.yml`, eg:

```
---
keys:
- https://example.com/public.key
repos:
- deb http://apt.example.com stable main
packages:
- ascii
- libxml
```

### Building the Buildpack

To build this buildpack, run the following command from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.

   ```bash
   source .envrc
   ```
   To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Install buildpack-packager

    ```bash
    (cd src/apt/vendor/github.com/cloudfoundry/libbuildpack/packager/buildpack-packager && go install)
    ```

1. Build the buildpack

    ```bash
    buildpack-packager
    ```

1. Use in Cloud Foundry

   Upload the buildpack to your Cloud Foundry and optionally specify it by name

    ```bash
    cf create-buildpack [BUILDPACK_NAME] [BUILDPACK_ZIP_FILE_PATH] 1
    cf push my_app [-b BUILDPACK_NAME]
    ```

### Testing

Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass) framework for running integration tests.

To test this buildpack, run the following command from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.

   ```bash
   source .envrc
   ```
   To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Run unit tests

    ```bash
    ./scripts/unit.sh
    ```

1. Run integration tests

    ```bash
    ./scripts/integration.sh
    ```

More information can be found on Github [cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass).

### Contributing

Find our guidelines [here](./CONTRIBUTING.md).

### Help and Support

Join the #buildpacks channel in our [Slack community](http://slack.cloudfoundry.org/)

### Reporting Issues

Open an issue on this project

### Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066)

## Disclaimer

This buildpack is experimental and not yet intended for production use.
