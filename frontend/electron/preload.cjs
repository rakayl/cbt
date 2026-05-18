const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('desktopApp', {
  isDesktop: true,
  platform: process.platform,
  enterExamMode: () => ipcRenderer.invoke('desktop:enter-exam-mode'),
  exitExamMode: () => ipcRenderer.invoke('desktop:exit-exam-mode'),
});
