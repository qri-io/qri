const {app, BrowserWindow} = require('electron')
var WebSocketClient = require('websocket').client;
var client = new WebSocketClient();
var socket = null;

// Keep a global reference of the window object, if you don't, the window will
// be closed automatically when the JavaScript object is garbage collected.
let win

function close() {
  if(socket != null) {
    socket.sendUTF(JSON.stringify({event:"shutdown", data: true }));
    socket.close()
  }
}

function createWindow () {
  // Create the browser window.
  win = new BrowserWindow({width: 1280, height: 720})

  // and load the index.html of the app.
  win.loadURL(`file://${__dirname}/assets/index.html`)

  // Open the DevTools.
  // win.webContents.openDevTools()

  // Emitted when the window is closed.
  win.on('closed', () => {
    // Dereference the window object, usually you would store windows
    // in an array if your app supports multi windows, this is the time
    // when you should delete the corresponding element.
    win = null
    close()
  })
}

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', createWindow)

// Quit when all windows are closed.
app.on('window-all-closed', () => {
  // On macOS it is common for applications and their menu bar
  // to stay active until the user quits explicitly with Cmd + Q
  close()
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('activate', () => {
  // On macOS it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (win === null) {
    createWindow()
  }
})

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and require them here.
const ipc = require('electron').ipcMain
ipc.on('asynchronous-message', function (event, arg) {
  // event.sender.send('asynchronous-reply', 'pong')
})

//Websocket
client.on('connectFailed', function(error) {
  console.log('Connect Error: ' + error.toString());
});

client.on('connect', function(connection) {
  console.log('WebSocket Client Connected');

  socket = connection; //copy to global scope

  connection.on('error', function(error) {
    console.log("Connection Error: " + error.toString());
  });

  connection.on('close', function() {
    console.log('Websockt Connection Closed');
    app.quit()
  });

  // connection.on('message', function(message) {
  //   if (message.type === 'utf8') {
  //     console.log("Received: '" + message.utf8Data + "'");
  //   }
  // });
});

client.connect('ws://127.0.0.1:9109/ui', []);
