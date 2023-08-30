# This is a simple script that cleans up the files created in the `many`
# experiment and restores the original `effects.txt` file. 
# The `benchmark` script does the cleanup, but if it is not executed fully,
# can run this script to do the cleanup.



#!/bin/bash

#SBATCH --mail-user=mashalimay@cs.uchicago.edu
#SBATCH --mail-type=ALL
#SBATCH --job-name=proj3_benchmark 
#SBATCH --output=./slurm/out/%j.%N.stdout
#SBATCH --error=./slurm/out/%j.%N.stderr
#SBATCH --chdir=/home/mashalimay/ParallelProgramming/project-3-mshalimay/proj3/
#SBATCH --partition=debug 
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=16
#SBATCH --mem-per-cpu=900
#SBATCH --exclusive
#SBATCH --time=4:00:00


module load golang/1.19

data_dir_in="./data/in/"
effects_dir="./data/"

data_dirs=("small" "mixture" "big")

many=4

for data_dir in "${data_dirs[@]}"; do
    find "${data_dir_in}${data_dir}/" -type f -name "*_[1-${many}].png" -delete
done

# restore the original effects.txt file
rm ${effects_dir}effects.txt
cp ${effects_dir}effects_backup.txt ${effects_dir}effects.txt
