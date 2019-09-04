import React from 'react';
import {eventDuration} from './googleEventTools'
import { List, Card } from 'antd'
import { Descriptions } from 'antd'
import PieChart from 'react-minimal-pie-chart';

var titleCase = (text) => {
  var result = text.replace( /([A-Z])/g, " $1" );
  return result.charAt(0).toUpperCase() + result.slice(1);
}

const colors = ["#c19160","#37a24f","#6697c3","#ffd00d","#fb8b00"]

class CalendarSummary extends React.Component {

  renderSummary(item) {
    let details = []
    for (const key of Object.keys(item)) {
      if (key !== "title")
      details.push(<Descriptions.Item key={key} label={titleCase(key)}>{item[key].toFixed(2)}</Descriptions.Item>)
    }

    return (
      <Descriptions layout="vertical" bordered>
        { details }
      </Descriptions>
    )
  }

  render() {

    let summary = []
    let attended = {

    }

    if (this.props.data && this.props.data.attended) {
      attended = {
        title: "attended",
        count: this.props.data.attended.length,
        totalHours: this.props.data.attended.reduce( (total, event) => {
          return total + eventDuration(event)
        }, 0)
      }

      Object.keys(this.props.data).forEach((key,index) => {
        if (key === "attended")
          return;

        let current = this.props.data[key]
        let totalHours = current.reduce( (total, event) => {
          return total + eventDuration(event)
        }, 0)
        summary.push({
          title: key,
          count: current.length,
          totalHours: totalHours,
          percentOfTotal: 100 * totalHours / attended.totalHours
        })
      });
    }
    let pieData = summary.filter( (item) => (item.title !== "attended") ).map( (item, index) => ({title: item.title, value: item.totalHours, color: colors[index]}))
    console.log(pieData)

    return (

        <div>
          <Card title={attended.title}>
            {this.renderSummary(attended)}
          </Card>
          <List
            grid={{ gutter: 16, column: 4 }}
            dataSource={summary}
            renderItem={ (item, index) => (
              <List.Item>
                <Card headStyle={ { "background-color": colors[index]} } title={item.title}>
                {this.renderSummary(item)}
                </Card>
              </List.Item>
            )}
          />

          <PieChart
            style={ {width:"30%"} }
            data={ pieData }
          />

        </div>
    );
  }
}

export default CalendarSummary;
