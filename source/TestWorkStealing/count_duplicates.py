import re


# load file from txt
with open("log.txt", "r") as file:
    data = file.read().splitlines()


duplicates = {}
count = 0
tasks_executed = 0
# count duplicates
for line in data:

    task_id = None
    
    # regex pattern
    pattern = r"Worker \d+ (executing task|exec task) (\d+) (on behalf of worker \d+|)"
    # regex match
    match = re.search(pattern, line)
    # if match found
    if match:
        tasks_executed += 1
        task_id = match.group(2)
        # if `ID` is not in duplicates dict
        if task_id not in duplicates:
            # add `ID` to duplicates dict
            duplicates[task_id] = 1
        else:
            # increment `ID` count
            duplicates[task_id] += 1
            count += 1

    if line == data[-1]:
        total_tasks = line


print(total_tasks)
print(f"Total duplicates: {count}")        
print(f"Total tasks executed: {tasks_executed}")

