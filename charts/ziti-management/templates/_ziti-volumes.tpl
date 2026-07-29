{{/*
These two names belong to the service-base library chart. Redefining them is
how this chart gets its identity volume into the shared deployment template,
which has no other extension point for it.

Helm keeps one template namespace for the whole release, so these definitions
replace the library's for *every* chart rendered alongside this one, not just
this chart. The ziti branch is therefore gated on the chart name: any other
chart falls through to output identical to the library's implementation, so
enabling persistence for its own volume cannot hand it a ziti volume it never
asked for.
*/}}

{{- define "ziti-management.wantsZitiVolume" -}}
{{- $persistence := .Values.persistence | default dict -}}
{{- if and $persistence.enabled (eq .Chart.Name "ziti-management") -}}
true
{{- end -}}
{{- end -}}

{{- define "service-base.renderExtraVolumes" -}}
{{- $volumes := list -}}
{{- if include "ziti-management.wantsZitiVolume" . }}
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
{{- if include "ziti-management.wantsZitiVolume" . }}
{{- $mounts = append $mounts (dict "name" "ziti-data" "mountPath" "/var/lib/ziti") -}}
{{- end }}
{{- with .Values.extraVolumeMounts }}
{{- $mounts = concat $mounts . }}
{{- end }}
{{- if $mounts }}
{{ toYaml $mounts }}
{{- end }}
{{- end -}}
