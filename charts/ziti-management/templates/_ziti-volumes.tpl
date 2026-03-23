{{- define "service-base.renderExtraVolumes" -}}
{{- $volumes := list -}}
{{- $persistence := .Values.persistence | default dict -}}
{{- if $persistence.enabled }}
{{- $claimName := printf "%s-ziti-data" (include "service-base.fullname" .) -}}
{{- $volumes = append $volumes (dict "name" "ziti-data" "persistentVolumeClaim" (dict "claimName" $claimName)) -}}
{{- end }}
{{- with .Values.extraVolumes }}
{{- $volumes = concat $volumes . }}
{{- end }}
{{- if $volumes }}
{{ toYaml $volumes }}
{{- end }}
{{- end -}}

{{- define "service-base.renderExtraVolumeMounts" -}}
{{- $mounts := list -}}
{{- $persistence := .Values.persistence | default dict -}}
{{- if $persistence.enabled }}
{{- $mounts = append $mounts (dict "name" "ziti-data" "mountPath" "/var/lib/ziti") -}}
{{- end }}
{{- with .Values.extraVolumeMounts }}
{{- $mounts = concat $mounts . }}
{{- end }}
{{- if $mounts }}
{{ toYaml $mounts }}
{{- end }}
{{- end -}}
