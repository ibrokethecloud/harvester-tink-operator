{{/*
Generate certificates for tinkerbell certs
*/}}

{{- define "tinkerbell-certs" -}}
{{- $altNames := list "tinkerbell.registry" "tinkerbell.tinkerbell" "tinkerbell" "localhost" "registry" "registry.default" "registry.default.svc"  "regisstry.default.svc.cluster.local" "tink-server" "tink-server.default" "tink-server.default.svc" "tink-server.default.svc.cluster.local" -}}
{{- $ca := genCA "custom-metrics-ca" 3650 -}}
{{- $cert := genSignedCert "tinkerbell" nil $altNames 3650 $ca -}}
tls.crt: {{ $cert.Cert | b64enc }}
tls.key: {{ $cert.Key | b64enc }}
ca.crt: {{ $ca.Cert | b64enc }}
{{- end -}}