Welcome to our Elevator Project!

We recommend looking at the diagrams, to make the structure of the project a little easier to grasp

Brief overview of the project structure:

Project is divided into elevator and node. The node is an entity on the network, communicating with other entities on the network (nodes)
One node also communicates with one elevator program (on its local machine)

We have a master-slave system. 
This is represented by the state of a node. See stateDiagram.png for more on this

We send a lot of messages!
A summary of the different types is found in the messageTypes.png 
Please also look at the Network/messages package, as this is where the messages are implemented, with some comments on their intended usage
Some of the data fields in the png are obsolete/have been changed

The messages are sent between different processes (go routines) 
I have attempted to create an overview of the channels connecting the different processes in channelDiagram.jpg
This only illustrates the channels in use while the node is in state Slave/master
IMPORTANT: When looking at the main "for-select" blocks in slave and master, please refer back to channelDiagram.jpg to get a better
idea of where the messages are actually going/coming from

Otherwise:
As you may notice, there is a lack of light control.
This logic is still missing, though some infrastructure exists for its implementation

We hope for some helpful feedback, thank you for reading!