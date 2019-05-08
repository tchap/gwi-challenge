#!/bin/sh

if [ -n "$GWI_ENTRYPOINT_PERFORM_DB_MIGRATION" ]; then
    echo '---> Executing database migrations'
    /gwi-api db migrate
    return_value="$?"
    if [ "$return_value" -ne 0 ]; then
        exit "$return_value"
    fi
fi

echo '---> Running the service executable'
exec /gwi-api $@