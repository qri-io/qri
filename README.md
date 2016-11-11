# Qri CLI


Qri should behave like npm & git combined, for data.

The datapackage.json file should be considered an integral part of the version-tracking structure, identifying sources of data & checking for data validity. Everything that isn't specified by the datapackage file should be considered a "script", and just tracked as blob data.

Commits should break out 3 types of changes:
	1. Schema Changes
	2. Data Changes
	3. Plaintext Changes


There are a few different concepts that need to be resolved before the qri cli will make proper sense:

### Migrations & Changes
This is the notion that there are two distinct types of changes in a dataset:
1. Changes to the way data is organized (schema changes)
2. Changes to the data in a dataset (dataset changes)

### Commits
This is the idea that a series of changes must be grouped together into a logical "save" that arrests changes into a record. In Git-land a "commit" marks a point in time & series of changes.

### DataPackages
This bit feels less connected to the first two problems, but is connected to the notion that the data package must be connected with t

### Query
In this world a query is nothing more than a computed dataset