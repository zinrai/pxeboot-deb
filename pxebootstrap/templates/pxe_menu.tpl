DEFAULT menu.c32
PROMPT 0
TIMEOUT 300

{{- range .Targets }}
LABEL {{ .Name }}_{{ .Codename }}_iso
    MENU LABEL ^Install {{ .Name }} {{ .Codename }}
    KERNEL memdisk
    INITRD http://{{ $.PXEServerHost }}/iso/images/{{ .Name }}/{{ .Codename }}/{{ base .ISOFile }}
    APPEND iso raw
{{- end }}
