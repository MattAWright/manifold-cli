# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.1.0] - 2017-07-19

### Added

- The start of time with the brand new `manifold-cli` tool!
- Intoducing the ability to login and out of a session through `manifold-cli
  login` and `manifold-cli logout`.
- Enabling a user to login using `MANIFOLD_EMAIL` and `MANIFOLD_PASSWORD`
  through any command.
- Allowing a user to export all credentials or only those for a specific app
  through `manifold-cli export`.
- Allowing a user to start a process by having Manifold inject the credentials
  directly into the process at startup through the `manifold-cli run` command.
- Enabling a user to provision a resource using the `manifold-cli create`
  command with a wizard or via a script using flags and arguments.
