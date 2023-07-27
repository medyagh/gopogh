// Displays an error message to the UI. Any previous message will be erased.
function displayError(message) {
    // Clear the body of all children.
    while (document.body.firstChild) {
      document.body.removeChild(document.body.firstChild);
    }
    const element = document.createElement("p");
    element.innerText = "Error: " + message;
    element.style.color = "red";
    element.style.fontFamily = "Arial";
    element.style.fontWeight = "bold";
    element.style.margin = "5rem";
    document.body.appendChild(element);
  }

// Parse URL search `query` into [{key, value}].
function parseUrlQuery(query) {
    if (query[0] === '?') {
      query = query.substring(1);
    }
    return Object.fromEntries((query === "" ? [] : query.split("&")).map(element => {
      const keyValue = element.split("=");
      return [unescape(keyValue[0]), unescape(keyValue[1])];
    }));
  }


function createRecentFlakePercentageTable(recentFlakePercentTable) {
    const createCell = (elementType, text) => {
      const element = document.createElement(elementType);
      element.innerHTML = text;
      return element;
    }
  
    const table = document.createElement("table");
    const tableHeaderRow = document.createElement("tr");
    tableHeaderRow.appendChild(createCell("th", "Rank"));
    tableHeaderRow.appendChild(createCell("th", "Test Name")).style.textAlign = "left";
    tableHeaderRow.appendChild(createCell("th", "Recent Flake Percentage"));
    tableHeaderRow.appendChild(createCell("th", "Growth (since last 15 days)"));
    table.appendChild(tableHeaderRow);
    const tableBody = document.createElement("tbody");
    for (let i = 0; i < recentFlakePercentTable.length; i++) {
        const { testName, recentFlakePercentage, growthRate } = recentFlakePercentTable[i];
        const row = document.createElement("tr");
        row.appendChild(createCell("td", "" + (i + 1))).style.textAlign = "center";
        row.appendChild(createCell("td", "" + testName));
        row.appendChild(createCell("td", recentFlakePercentage + "%")).style.textAlign = "right";
        row.appendChild(createCell("td", `<span style="color: ${growthRate === 0 ? "black" : (growthRate > 0 ? "red" : "green")}">${growthRate > 0 ? '+' + growthRate : growthRate}%</span>`));
      tableBody.appendChild(row);
    }
    table.appendChild(tableBody);
    new Tablesort(table);
    return table;
  }

  function displayTestAndEnvironmentChart(data) {
  }

  function displayEnvironmentChart(data) {
    const chartsContainer = document.getElementById('chart_div');
    chartsContainer.appendChild(createRecentFlakePercentageTable(data.recentFlakePercentTable))
  }

  async function init() {
    const query = parseUrlQuery(window.location.search);
    const desiredTest = query.test, desiredEnvironment = query.env || "", desiredPeriod = query.period || "";
  
    google.charts.load('current', { 'packages': ['corechart'] });
    try {
        // Wait for Google Charts to load
        await new Promise(resolve => google.charts.setOnLoadCallback(resolve));
    
        let url;
        if (desiredTest === undefined) {
          // URL for displayEnvironmentChart
          url = 'http://localhost:8080/env' + '?env=' + desiredEnvironment;
        } else {
          // URL for displayTestAndEnvironmentChart
          url = 'YOUR_TEST_AND_ENVIRONMENT_CHART_DATA_URL_HERE'; // Replace with the actual URL for test and environment chart data.
        }
    
        // Fetch data from the determined URL
        const response = await fetch(url);
        if (!response.ok) {
          throw new Error('Network response was not ok');
        }
        const data = await response.json();
        console.log(data)
    
        // Call the appropriate chart display function based on the desired condition
        if (desiredTest === undefined) {
          displayEnvironmentChart(data);
        } else {
          displayTestAndEnvironmentChart(data);
        }
      } catch (err) {
        displayError(err);
      }
  }
  
  init();