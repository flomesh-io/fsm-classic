#!/bin/bash

if ! git diff --exit-code ${SCRIPTS_TAR} ; then
    echo -e "\nPlease commit the changes made by 'make package-scripts'"
    exit 1
fi
