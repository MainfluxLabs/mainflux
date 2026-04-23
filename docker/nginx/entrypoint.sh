#!/bin/ash

if [ -z "$MF_MQTT_CLUSTER" ]
then
      envsubst '${MF_MQTT_ADAPTER_MQTT_PORT}' < /etc/nginx/snippets/mqtt-upstream-single.conf > /etc/nginx/snippets/mqtt-upstream.conf
      envsubst '${MF_MQTT_ADAPTER_WS_PORT}' < /etc/nginx/snippets/mqtt-ws-upstream-single.conf > /etc/nginx/snippets/mqtt-ws-upstream.conf
else
      envsubst '${MF_MQTT_ADAPTER_MQTT_PORT}' < /etc/nginx/snippets/mqtt-upstream-cluster.conf > /etc/nginx/snippets/mqtt-upstream.conf
      envsubst '${MF_MQTT_ADAPTER_WS_PORT}' < /etc/nginx/snippets/mqtt-ws-upstream-cluster.conf > /etc/nginx/snippets/mqtt-ws-upstream.conf
fi

envsubst '
    ${MF_USERS_HTTP_PORT}
    ${MF_THINGS_HTTP_PORT}
    ${MF_THINGS_AUTH_HTTP_PORT}
    ${MF_HTTP_ADAPTER_PORT}
    ${MF_MQTT_ADAPTER_HTTP_PORT}
    ${MF_NGINX_MQTT_PORT}
    ${MF_NGINX_MQTTS_PORT}
    ${MF_AUTH_HTTP_PORT}
    ${MF_WS_ADAPTER_PORT}
    ${MF_UI_PORT}
    ${MF_TIMESCALE_READER_PORT}
    ${MF_WEBHOOKS_HTTP_PORT}
    ${MF_SMTP_NOTIFIER_PORT}
    ${MF_DOWNLINKS_HTTP_PORT}
    ${MF_CONVERTERS_PORT}
    ${MF_FILESTORE_HTTP_PORT}
    ${MF_ALARMS_HTTP_PORT}
    ${MF_RULES_HTTP_PORT}
    ${MF_CERTS_HTTP_PORT}
    ${MF_MODBUS_HTTP_PORT}
    ${MF_UI_CONFIGS_HTTP_PORT}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf


# CRL support: if the CRL directory is mounted, wait for the initial CRL file
# from the certs service, then watch for changes and reload nginx.
CRL_FILE="/etc/ssl/certs/crl/crl.pem"
# Work on a copy so we never modify the mounted original.
cp /etc/nginx/snippets/ssl-client.conf /etc/nginx/snippets/ssl-client-active.conf
if [ -d "$(dirname "$CRL_FILE")" ]; then
    # Wait up to 60s for the certs service to generate the initial CRL file.
    echo "Waiting for CRL file at $CRL_FILE..."
    WAITED=0
    while [ ! -f "$CRL_FILE" ] && [ "$WAITED" -lt 60 ]; do
        sleep 2
        WAITED=$((WAITED + 2))
    done

    if [ ! -f "$CRL_FILE" ]; then
        echo "WARNING: CRL file not found after 60s. Starting without CRL."
        sed -i '/ssl_crl/d' /etc/nginx/snippets/ssl-client-active.conf
    else
        echo "CRL file found."
    fi

    # Watch for CRL updates and reload nginx.
    (
        LAST_MOD=""
        while true; do
            sleep 5
            if [ -f "$CRL_FILE" ]; then
                CURRENT_MOD=$(stat -c %Y "$CRL_FILE" 2>/dev/null || echo "")
                if [ -n "$CURRENT_MOD" ] && [ "$CURRENT_MOD" != "$LAST_MOD" ]; then
                    LAST_MOD="$CURRENT_MOD"
                    nginx -s reload 2>/dev/null || true
                fi
            fi
        done
    ) &
else
    echo "CRL directory not mounted. Starting without CRL."
    sed -i '/ssl_crl/d' /etc/nginx/snippets/ssl-client-active.conf
fi

exec nginx -g "daemon off;"
