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

echo "Memory per CPU: $SLURM_MEM_PER_CPU MB"

data_dir_in="./data/in/"
effects_dir="./data/"

# experiments=("few" "many") 
experiments=("few")  
many=3 # if experiment "many", experiment is ran for: (many * # images `in` folder)

# data directories, modes and set of threads to run
data_dirs=("small" "mixture" "big")

modes=("s" "parfiles" "parslices" "pipebsp" "pipebspws")

threads=("1" "2" "4" "6" "8" "10" "12" "14" "16")

# number of threads to slice each image into
subthreads=("1" "2" "4" "6" "8" "10")

# number of times to run the editor for each combination of data_dir, mode and thread
repeat=3

# Get start time
start=$(date +%s)

for experiment in "${experiments[@]}"; do
    echo "Running experiment: $experiment"

    # check if results.txt exists and remove it
    if [ -f ./benchmark/results.txt ]; then
        echo "Deleting old results.txt file"
        rm ./benchmark/results.txt
    fi

    # When the experiment is "many", create more data and modify effects.txt
    if [ "$experiment" == "many" ]; then
        # copy the images `many` times
        for data_dir in "${data_dirs[@]}"; do
            for file in ${data_dir_in}${data_dir}/*.png; do
                base=$(basename "$file")
                name=${base%.png}
                for i in $(seq 1 $many); do
                    cp "$file" "${data_dir_in}${data_dir}/${name}_${i}.png"
                done
            done
        done

        # Create new effects for each new image based off the originals

        # save the original effects.txt file 
        cp ${effects_dir}effects.txt ${effects_dir}effects_few.txt        # ovewrites

        # Add a sentinel line at the end of the input (necessary for jq to work correctly)
        echo "" >> ${effects_dir}effects.txt
        
        # Create the new effects_many.txt 
        > ${effects_dir}effects_many.txt
        temp_file=$(mktemp)
        for ((i=1; i<=$many; i++)); do
            while IFS= read -r line; do
                echo $line | jq -c --arg iter $i '.inPath |= sub(".png"; "_" + $iter + ".png") | .outPath |= sub("_Out.png"; "_" + $iter + "_Out.png")' >> $temp_file
            done < ${effects_dir}effects.txt
        done
        mv $temp_file ${effects_dir}effects_many.txt

        # Replace effects.txt with the effects_many.txt file
        cp ${effects_dir}effects_many.txt ${effects_dir}effects.txt
    fi

    # Loop over all combinations and run the go script
    for data_dir in "${data_dirs[@]}"; do
        for mode in "${modes[@]}"; do
            # sequential mode
            if [ "$mode" = "s" ]; then
                for ((i=1; i<=repeat; i++)); do
                    echo "Running: data_dir=$data_dir, mode=$mode, threads=1, iteration=$i"
                    go run ./editor/editor.go "$data_dir" "$mode" "1"
                done
            # parallel mode
            else
                for thread in "${threads[@]}"; do
                    # if mode is pipebspws or pipebsp then loop on subthreads
                    if [ "$mode" = "pipebspws" ] || [ "$mode" = "pipebsp" ]; then
                        for subthread in "${subthreads[@]}"; do
                            for ((i=1; i<=repeat; i++)); do
                                echo "Running: data_dir=$data_dir, mode=$mode, threads=$thread, subthreads=$subthread, iteration=$i"
                                go run ./editor/editor.go "$data_dir" "$mode" "$thread" "$subthread"
                            done
                        done
                    # if mode is not parfiles then loop on threads
                    else
                        for ((i=1; i<=repeat; i++)); do
                            echo "Running: data_dir=$data_dir, mode=$mode, threads=$thread, iteration=$i"
                            go run ./editor/editor.go "$data_dir" "$mode" "$thread"
                        done
                    fi
                done
            fi
        done
    done

    # copy results.txt to a file with the experiment name
    cat ./benchmark/results.txt >>./benchmark/results_${experiment}.txt


    # compute performance metrics and plot speedups
    go run ./benchmark/benchmark.go "$experiment"

    # Cleanup for 'many' experiment
    # delete the images created
    if [ "$experiment" = "many" ]; then
        for data_dir in "${data_dirs[@]}"; do
            find "${data_dir_in}${data_dir}/" -type f -name "*_[1-${many}].png" -delete
        done
        # restore the original effects.txt file
        rm ${effects_dir}effects.txt
        mv ${effects_dir}effects_few.txt ${effects_dir}effects.txt
    fi
done

# Get end time
end=$(date +%s)

# Calculate and print runtime
runtime=$((end-start))
minutes=$((runtime/60))
