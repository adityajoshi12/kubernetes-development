# Kubectl Plugins

### What’s a kubectl plugin?

Kubectl plugins are extensions of the kubectl client. Depending on which plugins you install, you will be able to take advantage of new commands and new complex functionality, which isn’t possible with the default install of kubectl.

### Key consideration
1. A plugin name must begins with `kubectl-`. 
2. To install a plugin, move its executable file to anywhere on your PATH.

### Discovering plugins 
kubectl provides a command kubectl plugin list that searches your `PATH` for valid plugin executables. Executing this command causes a traversal of all files in your `PATH`. Any files that are executable, and begin with kubectl- will show up in the order in which they are present in your PATH in this command's output. 

### Writing kubectl plugins
You can write a plugin in any programming language or script that allows you to write command-line commands.

I have added example plugin in following languages
1. [`bash` (shell script)](./bash/README.md)

### krew - a plugin manager for kubectl

Krew helps you find and install kubectl plugins made available in the community. Once plugins are installed with Krew, they’re automatically kept up to date by the plugin manager, making operating and managing your plugins that much easier.
