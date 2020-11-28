# Bluesound BluOS Controller Builder

This CLI tool will download the latest BluOS Controller software from
www.bluesound.com/downloads and then patch it for Linux and build a Snap
and AppImage package.

This is in response for the desire to be able to run the BluOS Controller
software on linux though Bluesound does not officially release a Linux version


## Prerequisites
In order for this CLI tool to work correctly, you **must** have NodeJS installed
on your system. You can use any method you desire, however since there are sevearl
methods that people choose to use based on their preference, this CLI takes a 
configuration parameter to the path of your chosen nodejs `bin` folder

The simplest way to install the latest stable version of NodeJS is to use NVM. Follow
the steps avaialble at the [NVM Install](https://github.com/nvm-sh/nvm#installing-and-updating) readme


Once NodeJS is installed on your system use the following command to find the path to
your NodeJS bin folder:

```
$ which npm
/home/david/.nvm/versions/node/v14.3.0/bin/npm
```

When configuring this cli tool, in the `config.yaml` file, place the path to the node bin folder.

Example config.yaml:

```yaml
nodeBinPath: "/home/david/.nvm/versions/node/v14.3.0/bin/"
```

In the future I may try to add an auto-detection of this. For now this was quick and easy.

## Running the tool

Once you have NodeJS set up, simply download the bs-patch tool from the Releases section
of this repository, create your config.yaml file, and then execute the tool:

1. Download the release
2. Extract the bs-patch binary
3. Execute `which npm`
4. Copy the path to the `bin` folder
5. Create a file named `config.yaml` in the same folder as the `bs-patch` binary
6. Execute `./bs-patch`
7. (hopefully) Enjoy your new BluOS Controller app on Linux

```
$ ./bs-patch 
INFO[0000] Reading config.yaml                          
INFO[0000] Finding Latest BluOS Controller              
INFO[0002] Downloading Latest BlueOS controller: https://www.bluesound.com/wp-content/uploads/2020/11/BluOS-Controller-3.12.1.dmg 
Downloading... 98 MB complete      
INFO[0008] Extracting BlueOS Controller DMG             
INFO[0010] Extracting ASAR                              
INFO[0011] Patching electron.js                         
INFO[0011] Patching app.js - Update Check               
INFO[0011] Patching app.js - Update Platform            
INFO[0011] Adding electron dependency                   
INFO[0028] Adding electron-builder dependency           
INFO[0038] Building Snap and AppImage                   
INFO[0046] Cleaning up 
```

A new folder named `dist` will be created and contain the Snap and AppImage files. Enjoy!
