const { app, BrowserWindow, Menu, globalShortcut, ipcMain, shell } = require('electron');
const path = require('path');

const isDev = !app.isPackaged;
let mainWindow;

function registerSecurityShortcuts() {
  if (isDev) return;
  ['CommandOrControl+R', 'F5', 'CommandOrControl+Shift+I', 'CommandOrControl+W'].forEach((shortcut) => {
    globalShortcut.register(shortcut, () => {});
  });
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1366,
    height: 768,
    minWidth: 1024,
    minHeight: 720,
    show: false,
    autoHideMenuBar: true,
    title: 'CBT Kampus',
    backgroundColor: '#f5f8fa',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      devTools: isDev,
    },
  });

  Menu.setApplicationMenu(null);

  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:\/\//i.test(url)) shell.openExternal(url);
    return { action: 'deny' };
  });

  if (isDev) {
    mainWindow.loadURL(process.env.ELECTRON_RENDERER_URL || 'http://localhost:5173');
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html'));
  }
}

app.whenReady().then(() => {
  createWindow();
  registerSecurityShortcuts();
});

ipcMain.handle('desktop:enter-exam-mode', () => {
  if (!mainWindow) return false;
  mainWindow.setFullScreen(true);
  mainWindow.setAlwaysOnTop(true, 'screen-saver');
  return true;
});

ipcMain.handle('desktop:exit-exam-mode', () => {
  if (!mainWindow) return false;
  mainWindow.setAlwaysOnTop(false);
  mainWindow.setFullScreen(false);
  return true;
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) createWindow();
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

app.on('will-quit', () => {
  globalShortcut.unregisterAll();
});
