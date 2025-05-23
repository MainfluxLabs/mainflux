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
    ${MF_POSTGRES_READER_PORT}
    ${MF_WEBHOOKS_HTTP_PORT}
    ${MF_SMTP_NOTIFIER_PORT}
    ${MF_DOWNLINKS_HTTP_PORT}
    ${MF_CONVERTERS_PORT}
    ${MF_FILESTORE_HTTP_PORT}
    ${MF_ALARMS_HTTP_PORT}
    ${MF_RULES_HTTP_PORT}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf


exec nginx -g "daemon off;"
