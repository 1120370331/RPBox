!macro RPBOX_REFRESH_SHORTCUT SHORTCUT_PATH
  ${If} ${FileExists} "${SHORTCUT_PATH}"
    CreateShortcut "${SHORTCUT_PATH}" "$INSTDIR\${MAINBINARYNAME}.exe" "" "$R9" 0
    !insertmacro SetLnkAppUserModelId "${SHORTCUT_PATH}"
  ${EndIf}
!macroend

!macro NSIS_HOOK_POSTINSTALL
  IfFileExists "$INSTDIR\${MAINBINARYNAME}.exe" rpbox_prepare_icon 0
  Goto rpbox_refresh_done

rpbox_prepare_icon:
  CopyFiles /SILENT "$INSTDIR\rpbox-icon.ico" "$INSTDIR\rpbox-icon-${VERSION}.ico"
  StrCpy $R9 "$INSTDIR\rpbox-icon-${VERSION}.ico"
  IfFileExists "$R9" 0 rpbox_use_exe_icon
  Goto rpbox_refresh_shortcuts

rpbox_use_exe_icon:
  StrCpy $R9 "$INSTDIR\${MAINBINARYNAME}.exe"

rpbox_refresh_shortcuts:
  WriteRegStr SHCTX "${UNINSTKEY}" "DisplayIcon" "$\"$R9$\""
  WriteRegStr SHCTX "Software\Classes\rpbox\DefaultIcon" "" "$\"$R9$\",0"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$DESKTOP\${PRODUCTNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$DESKTOP\${MAINBINARYNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$SMPROGRAMS\${PRODUCTNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$SMPROGRAMS\${MAINBINARYNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$SMPROGRAMS\$AppStartMenuFolder\${PRODUCTNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$SMPROGRAMS\$AppStartMenuFolder\${MAINBINARYNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$APPDATA\Microsoft\Internet Explorer\Quick Launch\User Pinned\TaskBar\${PRODUCTNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$APPDATA\Microsoft\Internet Explorer\Quick Launch\User Pinned\TaskBar\${MAINBINARYNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$APPDATA\Microsoft\Internet Explorer\Quick Launch\${PRODUCTNAME}.lnk"
  !insertmacro RPBOX_REFRESH_SHORTCUT "$APPDATA\Microsoft\Internet Explorer\Quick Launch\${MAINBINARYNAME}.lnk"
  System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
  IfFileExists "$SYSDIR\ie4uinit.exe" 0 +2
    ExecWait '"$SYSDIR\ie4uinit.exe" -show'

rpbox_refresh_done:
!macroend

!macro NSIS_HOOK_PREUNINSTALL
  Delete "$INSTDIR\rpbox-icon-*.ico"
!macroend
