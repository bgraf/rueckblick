console.log("howdy!");

async function fetchChanges() {
  let response = await fetch("/changed");

  console.log("received!");

  if (response.status == 502) {
    // Status 502 is a connection timeout error,
    // may happen when the connection was pending for too long,
    // and the remote server or a proxy closed it
    // let's reconnect
    await fetchChanges();
  } else if (response.status != 200) {
    // An error - let's show it
    console.log(response.statusText);
    // Reconnect in one second
    await new Promise(resolve => setTimeout(resolve, 1000));
    await fetchChanges();
  } else {
    // Get and show the message
    let message = await response.text();
    console.log("done");
    console.log(message);
    // Call subscribe() again to get the next message
    // await subscribe();
    await new Promise(resolve => setTimeout(resolve, 200));
    window.location.reload();
  }
}

fetchChanges();