DEFAULT menu.c32
PROMPT 0
TIMEOUT 300

{{- range .Targets }}
LABEL {{ .Name }}_{{ .Codename }}_image
    MENU LABEL ^Install {{ .Name }} {{ .Codename }} {{ .Version }} image
    KERNEL /images/{{ .Name }}/{{ .Codename }}/{{ .Version }}/{{ base .BootFiles.Vmlinuz }}
    IPAPPEND 2
    APPEND initrd=/images/{{ .Name }}/{{ .Codename }}/{{ .Version }}/{{ base .BootFiles.Initrd }} ip=dhcp

LABEL {{ .Name }}_{{ .Codename }}_iso
    MENU LABEL ^Install {{ .Name }} {{ .Codename }} {{ .Version }} iso
    KERNEL memdisk
    INITRD http://{{ $.PXEServerHost }}/iso/images/{{ .Name }}/{{ .Codename }}/{{ .Version }}/{{ base .ISOFile }}
    APPEND iso raw

{{- end }}
