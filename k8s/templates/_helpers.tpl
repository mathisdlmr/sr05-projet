{{- define "werewolf.labels" -}}
app: werewolf
app.kubernetes.io/name: werewolf
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
