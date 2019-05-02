#!/bin/bash

EMAIL='me@example.com'
PASSWORD='secret'
BASE_URL='localhost:8888/v1'
VOLUNTEERS="$BASE_URL/volunteers"
TEAMS="$BASE_URL/teams"

function get() {
    token="$1"
    shift
    curl -s -H "Authorization: Bearer $token" "$@" | jq .
}

function post() {
    local token="$1"
    shift
    curl -s -X POST -H "Authorization: Bearer $token" -H 'Content-Type: application/json' "$@" | jq .
}

function put() {
    local token="$1"
    shift
    curl -s -X PUT -H "Authorization: Bearer $token" -H 'Content-Type: application/json' "$@" | jq .
}

function delete() {
    local token="$1"
    shift
    curl -s -X DELETE -H "Authorization: Bearer $token" "$@"
}

# Create a new user
echo '---> Create a new volunteer account'
token=$(curl -s -X POST \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
    "$VOLUNTEERS/login" | jq -r .token)

# Get me
echo '---> Get me (i.e. the account just created)'
get "$token" "$VOLUNTEERS/me"

# Create a team
echo '---> Create a new team'
TEAM_ID="gophers"
post "$token" -d "{\"id\": \"gophers\", \"name\": \"The Gophers\"}" "$TEAMS"

# Get the team just created
echo '---> Get the team just created'
get "$token" "$TEAMS/$TEAM_ID"

# List team members
echo '---> List the team members'
get "$token" "$TEAMS/$TEAM_ID/members"

# Join the team (no token)
echo '---> Join the team (token missing)'
put "" "$TEAMS/$TEAM_ID/members/$EMAIL"

# Join the team
echo '---> Join the team'
put "$token" "$TEAMS/$TEAM_ID/members/$EMAIL"

# List team members, again
echo '---> List the team members, again'
get "$token" "$TEAMS/$TEAM_ID/members"

# Leave the team
echo '---> Leave the team'
delete "$token" "$TEAMS/$TEAM_ID/members/$EMAIL"

# List team members, again
echo '---> List the team members, again'
get "$token" "$TEAMS/$TEAM_ID/members"

# Create another account
EMAIL='alterego@example.com'
PASSWORD='secret'

echo '---> Create another account'
token=$(curl -s -X POST \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
    "$VOLUNTEERS/login" | jq -r .token)

# Join the team created by the other account
echo '---> Join the team created by the other account'
put "$token" "$TEAMS/$TEAM_ID/members/$EMAIL"

# List team members, again
echo '---> List the team members'
get "$token" "$TEAMS/$TEAM_ID/members"