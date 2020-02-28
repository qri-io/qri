#!/bin/bash

# capture the output of time. time normally prints to stderr, this routes the 
# output of time to the results variable
time_command () {
  exec 3>&1 4>&2
  results=$({ TIMEFORMAT='%U,%S,%P%%,%R' && time $@ 1>&3 2>&4; } 2>&1)
  exec 3>&- 4>&-
}

# create an example csv file. Each row is 1000 characters of data, including
# commas.
#   $1: number of rows the file should have
# sets $filename to output file
example_csv_file() {
  filename=example_$1_rows.csv
  # make a row that is exactly 1000 characters in length
  row_data="BA882B47-B26A-4E29-BFB4-XXXXXXXXXXXX,2000-01-01 00:00:01.000 UTC,2000-01-01 00:00:02.000 UTC,$(printf 'x%.0s' {1..907})"
  # print header
  echo "uuid,ingest,occurred,raw_data"  > $filename;
  # print rows
  for i in `seq $1`; do echo $row_data >> $filename; done;
}

run_bench_rows () {
  rows=$1
  name="example_${rows}_rows"
  mkdir -p $name
  cd $name

  example_csv_file $rows
  size_human=$(du -h $filename | awk '{print $1}' | tr -d 'M')

  echo "bench $name: init"
  time_command qri init --source-body-path $filename --format CSV --name $name
  echo "${rows},$size_human,init,$results" >> ../perf.csv

  echo "bench $name: save"
  time_command qri save
  echo "${rows},$size_human,save,$results" >> ../perf.csv

  echo "bench $name: get body"
  time_command qri get body > /dev/null
  echo "${rows},$size_human,get body,$results" >> ../perf.csv

  echo "bench $name: get meta"
  time_command qri get meta > /dev/null
  echo "${rows},$size_human,get meta,$results" >> ../perf.csv

  echo "bench $name: get remove all"
  time_command qri remove --all "me/$name" --keep-files
  echo "${rows},$size_human,remove all,$results" >> ../perf.csv

  cd ../
  rm -rf $name
}

echo "num_rows,size_mb,command,user,system,cpu,total" > perf.csv
run_bench_rows 10000
run_bench_rows 50000
run_bench_rows 100000
run_bench_rows 500000
run_bench_rows 1000000
