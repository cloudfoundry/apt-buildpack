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
truncatesources: true
cleancache: true
keys:
- https://example.com/public.key
repos:
- deb http://apt.example.com stable main
packages:
- ascii
- libxml
```
`truncatesources` as the name suggests truncates the sources.list file and puts just the entries specified in repos section. 
This maybe needed in environment where ubuntu public repos are blocked.

`cleancache` calls `apt-get clean` and `apt-get autoclean`. Useful to purge any cached content.

#### Using a PPA

It's possible to use a PPA, but you need to indicate the GPG key for the PPA and the full repo line, not just the PPA name.  

To locate this information, navigate to the PPA on Launchpad.  Expand where it says "Technical Details about this PPA".  See [this Stack Overflow post](https://askubuntu.com/questions/496495/can-a-ppa-repository-be-added-to-etc-apt-source-list#496529) if you're having trouble finding it.

Under that, select the correct version of Ubuntu from the drop down.  Then you can copy and paste the sources.list entries presented there under the `repos` block in `apt.yml`.  Beneath the sources.list entry, you'll a label named "Signing Key" and beneath that a link.  Click on the link.  On the page that loads, you should see one GPG key entry.  In the `bits/keyID` column, you'll see a link.  Right click on that and copy the link.  Paste that in under the `keys` block in your `apt.yml`.

You should now be able to install packages from that PPA.

### Building the Buildpack

To build this buildpack, run the following commands from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.

   ```bash
   source .envrc
   ```
   To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Install buildpack-packager

    ```bash
    go install github.com/cloudfoundry/libbuildpack/packager/buildpack-packager
    ```

1. Build the buildpack

    ```bash
    buildpack-packager build
    ```

1. Use in Cloud Foundry

   Upload the buildpack to your Cloud Foundry and optionally specify it by name

    ```bash
    cf create-buildpack [BUILDPACK_NAME] [BUILDPACK_ZIP_FILE_PATH] 1
    cf push my_app [-b BUILDPACK_NAME]
    ```

### Testing

Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/tree/master/cutlass) framework for running integration tests against Cloud Foundry. Before running the integration tests, you need to login to your Cloud Foundry using the [cf cli](https://github.com/cloudfoundry/cli):

 ```bash
 cf login -a https://api.your-cf.com -u name@example.com -p pa55woRD
 ```

Note that your user requires permissions to run `cf create-buildpack` and `cf update-buildpack`. To run the integration tests, run the following command from the buildpack's directory:

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

### Contributing

Find our guidelines [here](./CONTRIBUTING.md).

### Help and Support

Join the #buildpacks channel in our [Slack community](http://slack.cloudfoundry.org/).

### Reporting Issues

Open an issue on this project

### Active Development

The project backlog is on [Pivotal Tracker](https://www.pivotaltracker.com/projects/1042066).

## Disclaimer

This buildpack is experimental and not yet intended for production use.

