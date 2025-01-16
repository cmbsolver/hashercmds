#!/bin/bash

# Loop from 2 to 50
for i in $(seq 2 50); do
  ./createbytearrays/createbytearrays $i
done