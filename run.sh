
output=$(go run main.go /Users/developer/Documents/Tests/my-test-project)

actions(){
    echo '--------------------------------------BEGIN-----------------------------------------'
    while read -r line; do
        echo "$line"
        if echo "$line" | grep 'JCOCDIFFERENT'; then
            echo "Found it"
            exit
        fi
    done <<< "$output"
    echo '--------------------------------------DONE-----------------------------------------'
}

while : ; do
   actions
done
