# Image Manifest Specification

## Introduction


In any system with A/B updates mechanism to support reliable system update, the manifest file is crucial for specifying the update metadata, including partition information and update instructions. It describes the contents of the update package, the way it's applied, and how to switch between multiple images. 
The following are the key aspects of the any system update. This information is used prior to booting a system image in preparing the storage layout and selecting the system version to boot etc.

Key aspects of the A/B update manifest are :

### Partition Information:
The manifest specifies the partitions to be updated, their sizes, and other relevant metadata. 
### Update Instructions:
It defines how the update process works, including mount points, device mapper configurations, and any necessary post-installation scripts. 
### Slot Management:
The manifest guides the update process to select the appropriate slot (A or B) for updating and switching to the updated slot after the update is complete. 
### Dynamic Partitions:
For devices with dynamic partitions, the manifest includes additional information on how to handle groups and partitions. 
### Update Metadata:
The manifest contains metadata about the update, including the update's version number, type (full or delta), and any necessary conditions for the update to apply. 

## Image Manifest Format
The following is Image manifest's minimal format that image composition tool can output along with the created system Image.

Software_package_manifest {
  Size /* Size of the image */
  Package_Version /* Version of the package to which the module belongs */
  Signature /* Hash of the payload and a signature of the hash */
  CRC /* CRC of the image, including the above fields. */
}
