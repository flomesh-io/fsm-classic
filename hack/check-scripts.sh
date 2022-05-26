#!/bin/bash

NEW_MD5=$(md5sum ${SCRIPTS_TAR})
echo "New MD5 is: ${NEW_MD5}"
OLD_MD5=$(cat ${SCRIPTS_TAR_MD5})
echo "Old MD5 is: ${OLD_MD5}"

if [[ "${NEW_MD5}" != "${OLD_MD5}" ]]; then
    echo -e "\nPlease commit the changes made by 'make package-scripts'"
    exit 1
fi
