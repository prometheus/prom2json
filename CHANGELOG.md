## master / unreleased

### **Breaking changes**

### Changes

* [CHANGE]
* [FEATURE]
* [ENHANCEMENT]
* [BUGFIX]

## 1.4.2 / 2025-04-09

* [BUGFIX] Fix formatting of error messages. #190

_Besides the small fix above, this release comes with updated dependencies, and the pre-built binaries are using Go1.24.1, both to avoid potential security issues. (Note that this is just precaution. We do not know of any relevant vulnerabilities in v1.4.1.)_

## 1.4.1 / 2024-09-19

_There are no code changes in this release. It merely comes with updated dependencies, and the pre-built binaries are using Go1.23.1., both to avoid potential security issues. (Note that this is just precaution. We do not know of any relevant vulnerabilities in v1.4.0.)_

## 1.4.0 / 2024-07-11

* [FEATURE] Support native histograms. #169
* [FEATURE] Support float histograms (classic and native). #176

_This release also comes with updated dependencies, and the pre-built binaries are using Go1.22.5., both to avoid potential security issues. (Note that this is just precaution. We do not know of any relevant vulnerabilities in v1.3.3.)_

## 1.3.3 / 2023-05-25

_There are no code changes in this release. It merely comes with updated
dependencies, and the pre-built binaries are using Go1.20.4., both to avoid potential security issues. (Note that this is just precaution. We do not know of any relevant vulnerabilities in v1.3.2.)_

## 1.3.2 / 2022-10-07

_There are no code changes in this release. It merely comes with updated
dependencies, and the pre-built binaries are using Go1.19.2., both to avoid potential security issues. (Note that this is just precaution. We do not know of any relevant vulnerabilities in v1.3.1.)_

## 1.3.1 / 2022-04-19

_There are no code changes in this release. It merely comes with updated
dependencies, and the pre-built binaries are using Go1.18.1., both to avoid potential security issues. (Note that this is just precaution. We do not know of any relevant vulnerabilities in v1.3.0.)_

## 1.3.0 / 2019-12-21

* [ENHANCEMENT] Saner settings for the HTTP transport, based on the usual
  defaults, but with a ResponseHeaderTimeout of one minute. #72
* [BUGFIX] Close metric family channel in case of errors to prevent leaking a
  goroutine. #70

## 1.2.2 / 2019-07-23

* [FEATURE] Add ARM container images. #61
* [BUGFIX] Properly set the sum in a histogram. #65

## 1.2.1 / 2019-05-20

_No actual code changes. Only a fix of the CircleCI config to make Docker
images available again._

* [BUGFIX] Fix image upload to Docker Hub and Quay.

## 1.2.0 / 2019-05-17

### **Breaking changes**

Users of the `prom2json` package have to take into account that the interface
of `FetchMetricFamilies` has changed (to allow the bugfix below). For users of
the command-line tool `prom2json`, this is just an internal change without any
external visibility.

### Changes

* [FEATURE] Support timestamps.
* [BUGFIX] Do not create a new Transport for each call of `FetchMetricFamilies`.

## 1.1.0 / 2018-12-09

* [FEATURE] Allow reading from STDIN and file (in addition to URL).
* [ENHANCEMENT] Support Go modules for dependency management.
* [ENHANCEMENT] Update dependencies.

## 1.0.0 / 2018-10-20

Initial release
