; NSIS Installer for TG Assistant (tgtool)
; Build: makensis installer.nsi

!define PRODUCT_NAME "TG Assistant"
!define PRODUCT_VERSION "2.0.0"
!define PRODUCT_PUBLISHER "tgtool"
!define PRODUCT_EXE "tgtool.exe"

SetCompressor lzma

Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "tgtool-setup-${PRODUCT_VERSION}.exe"
InstallDir "$PROGRAMFILES64\TG Assistant"
RequestExecutionLevel admin

Section "Install"
  SetOutPath "$INSTDIR"

  File "tgtool.exe"
  CreateDirectory "$INSTDIR\sessions"
  CreateDirectory "$INSTDIR\data"
  CreateDirectory "$INSTDIR\logs"
  CreateDirectory "$INSTDIR\media"

  CreateShortCut "$DESKTOP\TG Assistant.lnk" "$INSTDIR\${PRODUCT_EXE}"
  CreateDirectory "$SMPROGRAMS\TG Assistant"
  CreateShortCut "$SMPROGRAMS\TG Assistant\TG Assistant.lnk" "$INSTDIR\${PRODUCT_EXE}"
  CreateShortCut "$SMPROGRAMS\TG Assistant\Uninstall.lnk" "$INSTDIR\uninstall.exe"

  WriteUninstaller "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TG Assistant" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TG Assistant" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TG Assistant" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TG Assistant" "Publisher" "${PRODUCT_PUBLISHER}"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\tgtool.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir /r "$INSTDIR\sessions"
  RMDir /r "$INSTDIR\data"
  RMDir /r "$INSTDIR\logs"
  RMDir /r "$INSTDIR\media"
  RMDir "$INSTDIR"

  Delete "$DESKTOP\TG Assistant.lnk"
  RMDir /r "$SMPROGRAMS\TG Assistant"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TG Assistant"
SectionEnd
