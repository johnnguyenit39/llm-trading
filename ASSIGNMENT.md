# This is Development - developer test

### Requirements:

- This test should be completed using golang / php (8.3+) / typescript (nodejs/deno)
- The rest of the stack is free of choice, if applicable, a framework should be used.
- In total you should not spend more than 4 hours on this assignment. The assignment does **not** need to be finished in this time.
- Commit and push your code periodically, at least every hour.

### Assignment

This is Development has been assigned with the maintenance of an e-commerce system.
Unfortunately this system has a number of problems:

1. The total prices of orders are not consistent with the prices of products.
1. There are a number of products missing in the dataset.

Your tasked to deliver a report to identify these problems.
The e-commerce system has a REST API that can be used to retrieve orders and products.
Unfortunately the implementation has some problems. Not all endpoints are available and the API itself is not very stable and has some overall design issue.
You should take this into account when creating your solution.

### Deliverables

- Code :)
- Report with:
  - A list of orders where the total prices are not consistent with the product prices and the reason for this.
  - A list of missing products.
  - Top 5 customers based on total number of products.
  - Top 5 customers based on total revenue.

Extra:

- Unit tests

### Dataset

- 1.000 customers
- Per customer between 1 to 10 orders
- Per order between 1 to 25 line items
- 100.000 products

### API

Base url: [https://tidtestapi.thisisdevelopment.nl/api](https://tidtestapi.thisisdevelopment.nl/api)

##### Orders

Retrieve customers and orders (10 per page):

`GET /orders`

**Params**

| Name| In | Type | Required |
|--|--|--|--|
| page | query | int | Yes |

**Return values**

| Value | Description |
|--|--|
| 200 | OK |
| 404 | Not found |
| 5xx | Server error |

##### Products

Retrieve 1 product:

`GET /products/[productID]`

**Params**

| Name| In | Type | Required |
|--|--|--|--|
| productID | path | int | Yes |

**Return values**

| Value | Description |
|--|--|
| 200 | OK |
| 404 | Not found |
| 5xx | Server error |
