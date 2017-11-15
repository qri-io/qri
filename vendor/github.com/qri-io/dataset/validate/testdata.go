package validate

// has lazy quotes
var rawText1 = `first_name,last_name,username,age
"Rob","Pike",rob, 100
Ken,Thompson,ken, 75.5
"Robert","Griesemer","gri", 100`

// has nonNumeric quotes and comma inside quotes on last line
var rawText2 = `"first_name","last_name","username","age"
"Rob","Pike","rob", 22
"Robert","Griesemer","gri", 100
"abc","def,ghi","jkl",1000`

// same as above but with spaces in last line
var rawText2b = `"first_name","last_name","username","age"
"Rob","Pike","rob", 22
"Robert","Griesemer","gri", 100
"abc", "def,ghi", "jkl", 1000`

// NOTE: technically this is valid csv and we should be catching this at an earlier filter
var rawText3 = `<html>
<body>
<table>
<th>
<tr>col</tr>
</th>
</table>
</body>
</html>`

var rawText4 = `<html>
<body>
<table>
<th>
<tr>Last Name, First</tr>
<tr>
</th>
</table>
</body>
</html>`
