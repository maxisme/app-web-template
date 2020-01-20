#!/bin/bash

REPO_NAME="app-web-template"

# move to parent git repo directory
SCRIPTPATH="$( cd "$(dirname "$0")" ; pwd -P )"
cd $SCRIPTPATH
cd ../

cp images