#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://127.0.0.1:8080}"
BASE_URL="${BASE_URL%/}"
COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-.env.compose}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

RESPONSE_BODY=""
RESPONSE_STATUS=""

fail() {
	printf '[fail] %s\n' "$1" >&2
	exit 1
}

ok() {
	printf '[ok] %s\n' "$1"
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		fail "missing required command: $1"
	fi
}

require_cmd curl
require_cmd jq
require_cmd docker

if ! docker compose version >/dev/null 2>&1; then
	fail "docker compose is unavailable"
fi

request() {
	local method="$1"
	local path="$2"
	local body="${3:-}"
	local token="${4:-}"
	local response_file="$TMP_DIR/response.json"
	local curl_args=(
		-sS
		-o "$response_file"
		-w "%{http_code}"
		-X "$method"
		"$BASE_URL$path"
	)

	if [[ -n "$body" ]]; then
		curl_args+=(
			-H "Content-Type: application/json"
			-d "$body"
		)
	fi

	if [[ -n "$token" ]]; then
		curl_args+=(
			-H "Authorization: Bearer $token"
		)
	fi

	if ! RESPONSE_STATUS="$(curl "${curl_args[@]}")"; then
		fail "$method $path request failed"
	fi

	RESPONSE_BODY="$(<"$response_file")"
}

expect_status() {
	local expected="$1"
	local label="$2"

	if [[ "$RESPONSE_STATUS" != "$expected" ]]; then
		fail "$label returned HTTP $RESPONSE_STATUS, want $expected. Body: $RESPONSE_BODY"
	fi

	ok "$label"
}

json_field() {
	local filter="$1"
	local label="$2"

	if ! jq -er "$filter" <<<"$RESPONSE_BODY"; then
		fail "failed to parse $label from response: $RESPONSE_BODY"
	fi
}

redis_get() {
	local key="$1"

	if [[ ! -f "$COMPOSE_ENV_FILE" ]]; then
		fail "missing $COMPOSE_ENV_FILE; set COMPOSE_ENV_FILE or run from a prepared Compose environment"
	fi

	if ! docker compose --env-file "$COMPOSE_ENV_FILE" ps redis >/dev/null 2>&1; then
		fail "redis service is unavailable through docker compose"
	fi

	docker compose --env-file "$COMPOSE_ENV_FILE" exec -T redis redis-cli --raw GET "$key" | tr -d '\r'
}

health_body="$(curl -fsS "$BASE_URL/healthz")" || fail "healthz request failed"
if [[ "$health_body" != "ok" ]]; then
	fail "healthz returned $health_body, want ok"
fi
ok "healthz"

suffix="$(date +%s)-$$"
email="smoke-$suffix@example.com"
password="secret123"
title="smoke article $suffix"
content="compose smoke verification $suffix"

register_payload="$(jq -n --arg email "$email" --arg password "$password" '{email: $email, password: $password}')"
request POST "/register" "$register_payload"
expect_status 201 "register user"
user_id="$(json_field '.id' 'registered user id')"

login_payload="$(jq -n --arg email "$email" --arg password "$password" '{email: $email, password: $password}')"
request POST "/login" "$login_payload"
expect_status 200 "login"
token="$(json_field '.token' 'login token')"

create_payload="$(jq -n --arg title "$title" --arg content "$content" '{title: $title, content: $content}')"
request POST "/articles" "$create_payload" "$token"
expect_status 201 "create article"
article_id="$(json_field '.id' 'created article id')"

publish_payload="$(jq -n --argjson article_id "$article_id" '{article_id: $article_id}')"
request POST "/articles/publish" "$publish_payload" "$token"
expect_status 200 "publish article"

request GET "/articles"
expect_status 200 "list published articles"
if ! jq -e --argjson article_id "$article_id" --arg title "$title" \
	'any(.[]; .id == $article_id and .title == $title and .state == "published")' <<<"$RESPONSE_BODY" >/dev/null; then
	fail "published article $article_id was not found in public list: $RESPONSE_BODY"
fi

request GET "/articles/$article_id"
expect_status 200 "get article detail"
if ! jq -e --argjson article_id "$article_id" --arg content "$content" \
	'.id == $article_id and .content == $content and .state == "published"' <<<"$RESPONSE_BODY" >/dev/null; then
	fail "article detail did not match created article: $RESPONSE_BODY"
fi

request GET "/articles/$article_id"
expect_status 200 "get article detail again"

views="$(redis_get "article:views:$article_id")"
if [[ ! "$views" =~ ^[0-9]+$ ]]; then
	fail "redis article view count is not numeric for article $article_id: $views"
fi
if (( views < 2 )); then
	fail "redis article view count for article $article_id is $views, want at least 2"
fi
ok "redis view counter"

request POST "/articles/publish" "$publish_payload" "$token"
expect_status 409 "duplicate publish returns 409"

update_payload="$(jq -n '{title: "updated after publish", content: "should conflict"}')"
request PUT "/me/articles/$article_id" "$update_payload" "$token"
expect_status 409 "update published article returns 409"

request DELETE "/me/articles/$article_id" "" "$token"
expect_status 204 "delete article"

request GET "/articles/$article_id"
expect_status 404 "deleted article detail returns 404"

request GET "/articles"
expect_status 200 "list published articles after delete"
if jq -e --argjson article_id "$article_id" 'any(.[]; .id == $article_id)' <<<"$RESPONSE_BODY" >/dev/null; then
	fail "deleted article $article_id was still found in public list: $RESPONSE_BODY"
fi
ok "deleted article hidden from public list"

request DELETE "/me/articles/$article_id" "" "$token"
expect_status 404 "duplicate delete returns 404"

printf '[ok] smoke completed for user %s and article %s\n' "$user_id" "$article_id"
