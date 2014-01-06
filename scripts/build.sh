#!/bin/bash
set -x
export PROJECT_NAME="go-elasticocean"
export GOROOT="/usr/local/go"
export GOPATH="$JENKINS_HOME/workspace/$PROJECT_NAME"
export GOBIN="$GOPATH/bin"
export PATH="$GOROOT/bin:$PATH"
export BRANCH_NAME="master"
export PACKAGE="deploy"
export TGZ="$PACKAGE\.tgz"
make
make install
