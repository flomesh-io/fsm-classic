#!/bin/bash

TEMP_DIR=$(mktemp -d)

tar -C ${TEMP_DIR} -zxvf ${SCRIPTS_TAR}
CNT=$(diff -qr ${TEMP_DIR}/scripts ${CHART_COMPONENTS_DIR}/scripts | wc -l)

if [[ ${CNT} -gt 0 ]]; then
    echo -e "\nPlease commit the changes made by 'make package-scripts'"
    exit 1
fi
