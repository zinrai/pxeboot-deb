#!ipxe

menu PXE Boot Menu
item --gap -- --------------------------------
{{- range .Targets }}
item {{ .Name }}_{{ .Codename }} Install {{ .Name | title }} {{ .Codename }}
{{- end }}
item shell iPXE Shell
item exit  Exit to BIOS
choose selected || goto shell

{{ range .Targets -}}
:{{ .Name }}_{{ .Codename }}
sanboot --no-describe --drive 0x81 http://{{ $.PXEServerHost }}/iso/images/{{ .Name }}/{{ .Codename }}/{{ base .ISOFile }} || goto failed
goto start
boot

{{- end }}
:shell
shell

:exit
exit

:failed
echo Boot failed
prompt Press any key to continue
goto start
