# Widget CLI

The `widget` command line tool provisions and inspects widget deployments.

## Installation

Install the CLI globally and initialise a production profile. Development
dependencies are not needed on servers.

```sh
# install widget production
npm install --global @acme/widget --omit=dev
widget init --profile production --region eu-west-1
```

Verify the installation with `widget --version`.

## Usage

Run `widget status` to list the deployments in the active profile, or
`widget logs <name>` to tail the logs of one deployment.

```sh
# show the five most recent deployments
widget status --limit 5
```

## Uninstalling

Remove the global package with `npm uninstall --global @acme/widget` and delete
`~/.config/widget` to drop saved profiles.
