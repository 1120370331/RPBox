!macro NSIS_HOOK_POSTINSTALL
  IfFileExists "$INSTDIR\rpbox.exe" rpbox_refresh_shortcuts 0
  Goto rpbox_refresh_done

rpbox_refresh_shortcuts:
  IfFileExists "$DESKTOP\RPBox.lnk" 0 +2
    CreateShortCut "$DESKTOP\RPBox.lnk" "$INSTDIR\rpbox.exe" "" "$INSTDIR\rpbox.exe" 0
  IfFileExists "$SMPROGRAMS\RPBox.lnk" 0 +2
    CreateShortCut "$SMPROGRAMS\RPBox.lnk" "$INSTDIR\rpbox.exe" "" "$INSTDIR\rpbox.exe" 0
  IfFileExists "$APPDATA\Microsoft\Internet Explorer\Quick Launch\User Pinned\TaskBar\RPBox.lnk" 0 +2
    CreateShortCut "$APPDATA\Microsoft\Internet Explorer\Quick Launch\User Pinned\TaskBar\RPBox.lnk" "$INSTDIR\rpbox.exe" "" "$INSTDIR\rpbox.exe" 0
  System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'
  IfFileExists "$SYSDIR\ie4uinit.exe" 0 +2
    ExecWait '"$SYSDIR\ie4uinit.exe" -show'

rpbox_refresh_done:
!macroend
