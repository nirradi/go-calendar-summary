import React from 'react';
// import { List } from 'antd';

class CalendarSummary extends React.Component {


  render() {

    return (
        <div>
          {JSON.stringify(this.props.data)}
        </div>
    );
  }
}

export default CalendarSummary;
